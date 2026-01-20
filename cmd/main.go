package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/fsnotify/fsnotify"
)

// TODO: make dynamic?
const MY_IDENTITY = "mtls-secret-updater"

// constant
var FILES_TO_WATCH = [3]string{
	"/certs/ca.crt",
	"/certs/tls.crt",
	"/certs/tls.key",
}

func tryPatchSecret(ctx context.Context, secretInterface v1.SecretInterface, name string, patchBytes []byte) error {
	patchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := secretInterface.Patch(patchCtx, name, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{
		FieldManager: MY_IDENTITY,
	})

	return err
}

func patchSecret(ctx context.Context, secretInterface v1.SecretInterface, secretName string) {
	if ctx.Err() != nil {
		return
	}

	log.Println("Patching secret")

	// Read the certificate files
	ca_crt, err := os.ReadFile("/certs/ca.crt")
	if err != nil {
		log.Printf("Error reading ca.crt: %v\n", err)
		return
	}
	tls_crt, err := os.ReadFile("/certs/tls.crt")
	if err != nil {
		log.Printf("Error reading tls.crt: %v\n", err)
		return
	}
	tls_key, err := os.ReadFile("/certs/tls.key")
	if err != nil {
		log.Printf("Error reading tls.key: %v\n", err)
		return
	}

	patchData := map[string]map[string]string{
		"stringData": {
			"ca.crt":  string(ca_crt),
			"tls.crt": string(tls_crt),
			"tls.key": string(tls_key),
		},
	}

	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		log.Printf("Error marshaling patch data: %v\n", err)
		return
	}

	// Ensure the patch is applied, retrying on failure
	for {
		err := tryPatchSecret(ctx, secretInterface, secretName, patchBytes)
		if err == nil {
			break
		}

		log.Printf("Error patching secret: %v. Will retry in 5s\n", err)

		select {
		case <-time.After(5 * time.Second):
			continue
		case <-ctx.Done():
			log.Println("Context cancelled, stopping patch attempts")
			return
		}
	}

	log.Println("Patched secret")
}

func main() {
	// Process setup

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	secretName, ok := os.LookupEnv("SECRET_NAME")
	if !ok {
		log.Fatal("SECRET_NAME env var not set")
	}

	secretNamespace, ok := os.LookupEnv("SECRET_NAMESPACE")
	if !ok {
		log.Fatal("SECRET_NAMESPACE env var not set")
	}

	secretInterface := clientset.CoreV1().Secrets(secretNamespace)

	// Set a watch for file changes

	log.Println("Setting up /certs watch")

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = watcher.Add("/certs")
	if err != nil {
		log.Fatal(err)
	}

	// Initial patch on startup

	var patchCtx context.Context
	var patchCancel context.CancelFunc

	// Check if files exist
	filesExist := true
	for _, file := range FILES_TO_WATCH {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			filesExist = false
			break
		}
	}

	if filesExist {
		log.Println("All files exist, patching secret")
		patchCtx, patchCancel = context.WithCancel(ctx)
		go func(ctx context.Context) {
			patchSecret(ctx, secretInterface, secretName)
		}(patchCtx)
	} else {
		log.Println("Not all files exist, skipping the initial patch")
	}

	log.Println("Watching /certs for changes")

	for {
		select {
		case <-ctx.Done():
			log.Println("Received shutdown signal, exiting...")
			if patchCancel != nil {
				patchCancel()
			}
			return // Graceful exit
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) && slices.Contains(FILES_TO_WATCH[:], event.Name) {
				log.Printf("Detected change in %s\n", event.Name)
				// Cancel any ongoing patch operation
				if patchCancel != nil {
					patchCancel()
				}
				// Create new context for this specific patch attempt, linked to main shutdown ctx
				patchCtx, patchCancel = context.WithCancel(ctx)

				go func(ctx context.Context) {
					// Sleep to debounce rapid changes
					// NOTE: this is *not only* because of 3 files being updated
					// but also because it so happens that there are:
					// (1) at filesystem level - multiple writes for the same file in the same update
					// (2) at spiffe-helper level - multiple updates for the same SVID generation
					// it all happens seemingly in a 30ms window
					// so we wait a bit more to be sure everything is settled
					// before we read the files
					// Sleep with context awareness
					select {
					case <-time.After(100 * time.Millisecond):
					case <-ctx.Done():
						return
					}
					patchSecret(ctx, secretInterface, secretName)
				}(patchCtx)
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Fatalln("Watcher error:", err)
		}
	}
}
