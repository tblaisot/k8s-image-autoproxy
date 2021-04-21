package cmd

import (
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	m "github.com/tblaisot/k8s-image-autoproxy/pkg/mutate"
)

var (
	proxy   string
	verbose bool
)

func handleRoot(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello %q", html.EscapeString(r.URL.Path))
}

func handleMutate(w http.ResponseWriter, r *http.Request) {

	log.Println("handle mutate ...")
	// read the body / request
	body, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		sendError(err, w)
		return
	}

	// mutate the request
	mutated, err := m.Mutate(body, m.Config{Proxy: proxy, Verbose: verbose})
	if err != nil {
		sendError(err, w)
		return
	}

	// and write it back
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(mutated)

	if err != nil {
		return
	}
}

func sendError(err error, w http.ResponseWriter) {
	log.Println(err)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, "%s", err)
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "k8s-image-autoproxy",
	Short: "Alter image name to replace docker.io to another repository",
	Long: `This tool alter container specs to prefix images from docker.io with a custom proxy url
	
Usage:
  k8s-image-autoproxy --proxy proxy.io`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {

		log.Println("Starting server ...")

		mux := http.NewServeMux()

		mux.HandleFunc("/", handleRoot)
		mux.HandleFunc("/mutate", handleMutate)

		s := &http.Server{
			Addr:           ":8443",
			Handler:        mux,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20, // 1048576
		}

		log.Fatal(s.ListenAndServeTLS("/etc/webhook/certs/cert.pem", "/etc/webhook/certs/key.pem"))
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&proxy, "proxy", "p", "", "Proxy hostname to prefix all docker.io images.")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Print debug logs.")

	_ = rootCmd.MarkFlagRequired("proxy")
}
