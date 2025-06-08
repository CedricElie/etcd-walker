package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time" // Added for time.Now()

	"bazil.org/fuse"
	"bazil.org/fuse/fs"

	// Ensure these Kubernetes imports are present and correctly aliased
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1" // Aliased as metav1
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// Global variables for Kubernetes client and etcd pod details
var (
	k8sClientset      *kubernetes.Clientset
	k8sConfig         *rest.Config
	etcdPodName       string
	etcdNamespace     string
	etcdContainerName string
)

// execEtcdctlCommand is a wrapper to execute etcdctl commands
func execEtcdctlCommand(commandArgs []string) (stdout, stderr bytes.Buffer, err error) {
	// Prepend "etcdctl" to the commandArgs to form the full command
	fullCommand := append([]string{"etcdctl"}, commandArgs...)

	// Add static etcdctl connection params inside the pod
	fullCommand = append(fullCommand,
		"--endpoints=https://127.0.0.1:2379",
		// Using apiserver-etcd-client certs for client authentication
		"--cacert=/etc/kubernetes/pki/etcd/ca.crt",
		"--cert=/etc/kubernetes/pki/etcd/server.crt",
		"--key=/etc/kubernetes/pki/etcd/server.key",
	)

	log.Printf("Executing etcdctl inside pod: %s %s\n", etcdPodName, strings.Join(fullCommand, " "))

	return ExecCommandInPod(
		k8sClientset,
		k8sConfig,
		etcdNamespace,
		etcdPodName,
		etcdContainerName,
		fullCommand,
		nil, // No stdin
	)
}

// EtcdFS implements bazil.org/fuse/fs.FS for our FUSE filesystem
type EtcdFS struct {
	// No caching for now, could be added later
}

// Root returns the root directory of our filesystem.
func (efs EtcdFS) Root() (fs.Node, error) {
	return EtcdDir{Path: "/"}, nil
}

// EtcdDir represents a directory in our FUSE filesystem.
type EtcdDir struct {
	Path string // The etcd path this directory represents, always starts and ends with "/"
}

// Attr sets the attributes for a directory.
func (ed EtcdDir) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = os.ModeDir | 0o555 // Read-only directory permissions
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	return nil
}

// Lookup finds a child node (file or directory) within this directory.
func (ed EtcdDir) Lookup(ctx context.Context, name string) (fs.Node, error) {
	lookupPath := filepath.Join(ed.Path, name)

	// Try to get keys that start with lookupPath/ (indicating a directory)
	stdout, _, err := execEtcdctlCommand([]string{"get", lookupPath + "/", "--prefix", "--keys-only", "--limit=1"})
	if err == nil && strings.TrimSpace(stdout.String()) != "" {
		return EtcdDir{Path: lookupPath + "/"}, nil
	}

	// Try to get the exact key as a file
	stdout, stderr, err := execEtcdctlCommand([]string{"get", lookupPath})
	if err == nil && strings.TrimSpace(stdout.String()) != "" {
		lines := strings.SplitN(strings.TrimSpace(stdout.String()), "\n", 2)
		if len(lines) >= 2 {
			return EtcdFile{Path: lookupPath, Content: []byte(lines[1])}, nil
		}
		return EtcdFile{Path: lookupPath, Content: []byte{}}, nil
	} else if err != nil {
		if strings.Contains(strings.ToLower(stderr.String()), "not found") {
			return nil, fuse.ENOENT
		}
		log.Printf("etcdctl get error for %s: %v, Stderr: %s\n", lookupPath, err, stderr.String())
		return nil, fmt.Errorf("etcd lookup error: %w", err)
	}

	return nil, fuse.ENOENT // Not found
}

// ReadDirAll lists the contents of this directory.
func (ed EtcdDir) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	stdout, stderr, err := execEtcdctlCommand([]string{"get", ed.Path, "--prefix", "--keys-only"})
	if err != nil {
		log.Printf("etcdctl get --prefix error for %s: %v, Stderr: %s\n", ed.Path, err, stderr.String())
		return nil, fmt.Errorf("failed to read directory from etcd: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return nil, nil
	}

	entries := make(map[string]fuse.Dirent)
	for _, line := range lines {
		key := strings.TrimSpace(line)
		if key == "" {
			continue
		}

		relPath := strings.TrimPrefix(key, ed.Path)
		if relPath == "" {
			continue
		}

		parts := strings.Split(relPath, "/")
		name := parts[0]

		if name == "" {
			continue
		}

		if len(parts) > 1 && parts[1] != "" {
			entries[name] = fuse.Dirent{Name: name, Type: fuse.DT_Dir}
		} else {
			entries[name] = fuse.Dirent{Name: name, Type: fuse.DT_File}
		}
	}

	var dirents []fuse.Dirent
	for _, entry := range entries {
		dirents = append(dirents, entry)
	}
	return dirents, nil
}

// EtcdFile represents a file in our FUSE filesystem.
type EtcdFile struct {
	Path    string
	Content []byte // Content is pre-fetched by Lookup for simplicity, but ReadAll will re-fetch.
}

// Attr sets the attributes for a file.
func (ef EtcdFile) Attr(ctx context.Context, a *fuse.Attr) error {
	a.Mode = 0o444 // Read-only file permissions
	a.Size = uint64(len(ef.Content))
	a.Mtime = time.Now()
	a.Ctime = time.Now()
	return nil
}

// ReadAll reads the entire content of the file.
func (ef EtcdFile) ReadAll(ctx context.Context) ([]byte, error) {
	stdout, stderr, err := execEtcdctlCommand([]string{"get", ef.Path, "--print-value-only"})
	if err != nil {
		log.Printf("etcdctl get content error for %s: %v, Stderr: %s\n", ef.Path, err, stderr.String())
		if strings.Contains(strings.ToLower(stderr.String()), "not found") {
			return nil, fuse.ENOENT
		}
		return nil, fmt.Errorf("failed to read file content from etcd: %w", err)
	}
	return bytes.TrimSpace(stdout.Bytes()), nil // Trim trailing newline from etcdctl output
}

// The main function, likely in explore_etcd.go as per your error.
func main() {
	// --- Kubernetes Client Setup ---
	var config *rest.Config
	var err error

	config, err = rest.InClusterConfig()
	if err != nil {
		kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if _, statErr := os.Stat(kubeconfigPath); os.IsNotExist(statErr) {
			log.Fatalf("kubeconfig file not found at %s. Ensure it exists or run inside cluster.", kubeconfigPath)
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	k8sClientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes clientset: %v", err)
	}
	k8sConfig = config

	// --- Find the Etcd Pod ---
	etcdNamespace = "kube-system"
	etcdLabels := "component=etcd,tier=control-plane"

	log.Printf("Searching for etcd pod in namespace '%s' with labels '%s'...\n", etcdNamespace, etcdLabels)

	pods, err := k8sClientset.CoreV1().Pods(etcdNamespace).List(context.Background(), metav1.ListOptions{ // Corrected metav1
		LabelSelector: etcdLabels,
	})
	if err != nil {
		log.Fatalf("Failed to list etcd pods: %v", err)
	}

	if len(pods.Items) == 0 {
		log.Fatalf("No etcd pods found with labels '%s' in namespace '%s'.", etcdLabels, etcdNamespace)
	}

	etcdPod := pods.Items[0]
	etcdPodName = etcdPod.Name

	if len(etcdPod.Spec.Containers) == 0 {
		log.Fatalf("Etcd pod %s has no containers.", etcdPodName)
	}
	etcdContainerName = etcdPod.Spec.Containers[0].Name

	log.Printf("Found etcd pod: %s (container: %s)\n", etcdPodName, etcdContainerName)

	// --- FUSE Mount Setup ---
	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s <mountpoint>", os.Args[0])
	}
	mountpoint := os.Args[1]

	log.Printf("Mounting etcd-fs at %s\n", mountpoint)

	if err := os.MkdirAll(mountpoint, 0o755); err != nil {
		log.Fatalf("Failed to create mountpoint %s: %v", mountpoint, err)
	}
	entries, err := os.ReadDir(mountpoint)
	if err != nil {
		log.Fatalf("Failed to read mountpoint %s: %v", mountpoint, err)
	}
	if len(entries) > 0 {
		log.Fatalf("Mountpoint %s is not empty. Please use an empty directory.", mountpoint)
	}

	// Mount the FUSE filesystem
	// fuse.LocalVolume() is replaced by fuse.AllowOther() or fuse.DefaultPermissions()
	// depending on desired behavior. Default to AllowOther for broad access.
	// You might want to use fuse.ReadOnly() instead of fuse.AllowOther() if you don't need access from other users.
	c, err := fuse.Mount(
		mountpoint,
		fuse.FSName("etcd-fs"),
		fuse.Subtype("etcdctl-fuse"),
		fuse.ReadOnly(),   // Recommended for a read-only filesystem
		fuse.AllowOther(), // Allows other users to access the mountpoint (requires user_allow_other in /etc/fuse.conf)
		// Or if you only want the user running the process to access it:
		// fuse.DefaultPermissions(),
	)
	if err != nil {
		log.Fatalf("Failed to mount FUSE filesystem: %v", err)
	}
	defer c.Close()

	// Serve the FUSE filesystem
	err = fs.Serve(c, EtcdFS{})
	if err != nil {
		log.Fatalf("Failed to serve FUSE filesystem: %v", err)
	}

	// Wait for the mount process to complete or error.
	// The `c.Done()` and `c.Err()` methods on *fuse.Conn were removed.
	// Instead, fs.Serve blocks until the connection is closed or an error occurs.
	// You just need to check the error returned by fs.Serve.

	log.Println("Etcd-FUSE filesystem unmounted successfully.")
}

// ExecCommandInPod function remains unchanged, moved here for completeness.
func ExecCommandInPod(
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace, podName, containerName string,
	command []string,
	stdin io.Reader,
) (stdout, stderr bytes.Buffer, err error) {
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{ // Corrected corev1
			Command:   command,
			Container: containerName,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return stdout, stderr, fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	err = executor.StreamWithContext(context.Background(), remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return stdout, stderr, fmt.Errorf("failed to stream command output: %w", err)
	}
	return stdout, stderr, nil
}
