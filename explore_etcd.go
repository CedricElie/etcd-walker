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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// ExecCommandInPod executes a command in a pod and returns stdout and stderr.
// It's a reusable function from the previous example.
func ExecCommandInPod(
	clientset *kubernetes.Clientset,
	config *rest.Config,
	namespace, podName, containerName string, // containerName can be empty to use the first container
	command []string,
	stdin io.Reader,
) (stdout, stderr bytes.Buffer, err error) {
	// Build the URL for the exec request
	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Command:   command,
			Container: containerName, // Use the provided container name
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return stdout, stderr, fmt.Errorf("failed to create SPDY executor: %w", err)
	}

	// Stream the command's output
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

func main() {

	// --- Kubernetes Client Setup ---
	var config *rest.Config
	var err error

	// Try to load in-cluster config (if running inside a pod)
	config, err = rest.InClusterConfig()
	if err != nil {
		// Fallback to kubeconfig file (if running outside a cluster)
		kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
		if _, statErr := os.Stat(kubeconfigPath); os.IsNotExist(statErr) {
			log.Fatalf("kubeconfig file not found at %s. Ensure it exists or run inside cluster.", kubeconfigPath)
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			log.Fatalf("Error building kubeconfig: %v", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("Error creating Kubernetes clientset: %v", err)
	}

	// --- 1. Find the Etcd Pod ---
	etcdNamespace := "kube-system"
	etcdLabels := "component=etcd,tier=control-plane"

	log.Printf("Searching for etcd pod in namespace '%s' with labels '%s'...\n", etcdNamespace, etcdLabels)

	pods, err := clientset.CoreV1().Pods(etcdNamespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: etcdLabels,
	})
	if err != nil {
		log.Fatalf("Failed to list etcd pods: %v", err)
	}

	if len(pods.Items) == 0 {
		log.Fatalf("No etcd pods found with labels '%s' in namespace '%s'.", etcdLabels, etcdNamespace)
	}

	etcdPod := pods.Items[0] // Assuming there's at least one and we take the first one
	etcdPodName := etcdPod.Name

	// Usually, the etcd pod has only one container, or the etcd container is the first one.
	// You might want to explicitly check pod.Spec.Containers for "etcd" if multiple containers exist.
	etcdContainerName := "" // etcd typically has only one container, so leaving this empty will default to the first.
	if len(etcdPod.Spec.Containers) > 0 {
		etcdContainerName = etcdPod.Spec.Containers[0].Name
	}
	if etcdContainerName == "" {
		log.Fatalf("Could not determine container name for etcd pod %s.", etcdPodName)
	}

	log.Printf("Found etcd pod: %s (container: %s)\n", etcdPodName, etcdContainerName)

	// --- 2. Define the etcdctl command to execute ---
	// Make sure the etcdctl command is available inside the container's PATH.
	// For Kubeadm etcd pods, 'etcdctl' is usually directly available and pre-configured.
	etcdctlCommand := []string{
		"etcdctl",
		"--endpoints=https://127.0.0.1:2379",
		"--cacert=/etc/kubernetes/pki/etcd/ca.crt",
		"--cert=/etc/kubernetes/pki/etcd/server.crt",
		"--key=/etc/kubernetes/pki/etcd/server.key",
		"get",
		"/",
		"--prefix",
		"--keys-only",
	}
	// Or for health check:
	// etcdctlCommand := []string{"etcdctl", "endpoint", "health"}

	// --- 3. Execute the command inside the etcd pod ---
	fmt.Printf("Executing etcdctl command '%s' inside pod '%s/%s'...\n",
		strings.Join(etcdctlCommand, " "), etcdNamespace, etcdPodName)

	stdout, stderr, execErr := ExecCommandInPod(
		clientset,
		config,
		etcdNamespace,
		etcdPodName,
		etcdContainerName,
		etcdctlCommand,
		nil, // No stdin for this command
	)

	if execErr != nil {
		log.Fatalf("Error executing command in etcd pod: %v\nStdout: %s\nStderr: %s",
			execErr, stdout.String(), stderr.String())
	}

	fmt.Printf("Etcdctl command executed successfully.\n")
	if stdout.Len() > 0 {
		fmt.Printf("Etcd Keys:\n%s", stdout.String())
	}
	if stderr.Len() > 0 {
		fmt.Printf("Etcdctl Stderr (if any):\n%s", stderr.String())
	}
}
