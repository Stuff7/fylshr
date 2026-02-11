package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"mime"
	"net"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"time"
)

func main() {
	args := parseArgs()
	fs := http.FileServer(http.Dir(args.folder))

	var handler http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		url := r.URL.Path
		isDir := url[len(url)-1] == '/'

		if !isDir {
			filename := path.Base(url)
			if isMedia(filename) {
				disposition := fmt.Sprintf("attachment; filename=%s", strconv.Quote(filename))
				w.Header().Set("Content-Disposition", disposition)
			}
		}

		fs.ServeHTTP(w, r)

		if isDir {
			io.WriteString(w, style)
		}

		if !args.silent {
			fmt.Printf(
				"\x1b[1m\x1b[38;5;228m%s %s \x1b[38;5;195m%s\x1b[0m \x1b[38;5;225m%s\x1b[0m | \x1b[38;5;158m%s\x1b[0m\n",
				r.Method,
				r.URL.Path,
				r.Proto,
				r.RemoteAddr,
				r.Header.Get("User-Agent"),
			)
		}
	}

	fmt.Printf(
		"\x1b[1m\x1b[38;5;159mhttp://localhost:%s\n\x1b[38;5;158mhttp://%s:%s\n\x1b[38;5;225mCtrl-C\x1b[0m to exit\n",
		args.port,
		getLocalAddr(),
		args.port,
	)

	if args.https {
		fmt.Printf("\x1b[38;5;159mhttps://localhost:%s\n\x1b[38;5;158mhttps://%s:%s\n", args.port, getLocalAddr(), args.port)
		tlsConfig := generateTLSConfig()
		server := &http.Server{
			Addr:      ":" + args.port,
			Handler:   handler,
			TLSConfig: tlsConfig,
		}
		log.Fatal(server.ListenAndServeTLS("", "")) // certs provided by TLSConfig
	} else {
		log.Fatal(http.ListenAndServe(":"+args.port, handler))
	}
}

type Args struct {
	port   string
	folder string
	silent bool
	https  bool
}

func parseArgs() Args {
	port := flag.Int("port", 1080, "Port to listen")
	folder := flag.String("folder", "public", "Folder to serve")
	silent := flag.Bool("silent", false, "Do not log requests")
	https := flag.Bool("https", false, "Enable HTTPS")
	flag.Parse()

	return Args{
		port:   strconv.Itoa(*port),
		folder: *folder,
		silent: *silent,
		https:  *https,
	}
}

func isMedia(filename string) bool {
	extension := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(extension)

	switch mimeType {
	case
		"image/jpeg",
		"image/png",
		"image/gif",
		"video/mp4",
		"video/mpeg",
		"video/webm",
		"video/quicktime",
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-powerpoint",
		"application/vnd.openxmlformats-officedocument.presentationml.presentation",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"audio/mpeg",
		"audio/wav",
		"audio/ogg",
		"audio/midi",
		"application/ogg",
		"application/x-7z-compressed",
		"application/zip",
		"application/x-rar-compressed",
		"application/x-tar",
		"application/x-bzip2",
		"application/x-gzip",
		"application/x-zip-compressed",
		"application/x-tar-gz",
		"application/x-compressed-tar":
		return true
	default:
		return false
	}
}

func getLocalAddr() string {
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

// Generate in-memory self-signed TLS certificate
func generateTLSConfig() *tls.Config {
	priv, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)

	notBefore := time.Now()
	notAfter := notBefore.Add(365 * 24 * time.Hour)

	serialNumber, _ := rand.Int(rand.Reader, big.NewInt(1<<62))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    notBefore,
		NotAfter:     notAfter,
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	derBytes, _ := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	cert := tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  priv,
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

const style = `
<style>
  body {
    background: #111;
    color: #def;
  }

  *, *::before, *::after {
    font: 20px JetBrainsMono, mono, Menlo-Regular;
    box-sizing: border-box;

    scrollbar-width: thin;
    scrollbar-color: #aef #0003;
  }
  *::-webkit-scrollbar-thumb {
    background: #aef;
    border-radius: 20rem;
  }
  *::-webkit-scrollbar-track {
    background: #0003;
  }
  *::-webkit-scrollbar {
    width: 3rem;
  }

  pre {
    margin: 0;
    padding: 0.5rem;
  }

  a {
    color: #abf;
    font-weight: bold;
  }

  a:visited {
    color: #fba;
  }

  a:hover {
    color: #aef;
  }
</style>
`
