package main

import (
	"embed"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/adampresley/ftpslurper/internal/configuration"
	"github.com/adampresley/ftpslurper/internal/helpers"
	"github.com/adampresley/ftpslurper/internal/rendering"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

type Route struct {
	Path    string
	Method  string
	Handler http.HandlerFunc
}

type SFTPRequestHandler struct {
}

const (
	privateKeyPath     = "keys/serverKey/serverKey"
	authorizedKeysPath = "keys/authorizedKeys/authorizedKeys"
)

var (
	Version string = "development"
	appName string = "ftpslurper"

	//go:embed app
	appFS embed.FS

	//go:embed templates/*
	templateFS embed.FS

	headersOk = gorillahandlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"})
	originsOk = gorillahandlers.AllowedOrigins([]string{"*"})
	methodsOk = gorillahandlers.AllowedMethods([]string{"GET", "HEAD", "POST", "PUT", "OPTIONS"})

	/*
	 * Services
	 */
	httpHelpers      helpers.HttpHelpers
	templateRenderer rendering.TemplateRenderer

	/*
	 * Handlers
	 */

	/*
	 * Errors
	 */
	ErrUnauthorizedUser = fmt.Errorf("unauthorized user")
)

func main() {
	var (
		err error
		// db *gorm.DB
		privateSigner  ssh.Signer
		authorizedKeys map[string]bool
		listener       net.Listener
	)

	config := configuration.LoadConfig()

	setupLogger(&config, Version)

	slog.Info("configuration loaded",
		slog.String("version", Version),
		slog.String("loglevel", config.LogLevel),
		slog.String("address", config.Address),
	)

	slog.Debug("setting up database, services, and handlers...")

	if privateSigner, err = loadPrivateKey(privateKeyPath); err != nil {
		slog.Error("could not load private key", "error", err)
		os.Exit(1)
	}

	if authorizedKeys, err = loadAuthorizedKeys(authorizedKeysPath); err != nil {
		slog.Error("could not load authorized keys", "error", err)
		os.Exit(1)
	}

	ftpServerConfig := &ssh.ServerConfig{
		PublicKeyCallback: publicKeyAuth(authorizedKeys),
	}

	ftpServerConfig.AddHostKey(privateSigner)

	if listener, err = net.Listen("tcp", config.FTPAddress); err != nil {
		slog.Error("could not start FTP listener", "error", err, "address", config.FTPAddress)
		os.Exit(1)
	}

	defer listener.Close()

	slog.Info("FTP server started", "address", config.FTPAddress)

	for {
		conn, err := listener.Accept()

		if err != nil {
			slog.Error("could not accept connections", "error", err)
			os.Exit(1)
		}

		sshConn, chans, reqs, err := ssh.NewServerConn(conn, ftpServerConfig)

		if err != nil {
			if errors.Is(err, ErrUnauthorizedUser) {
				slog.Error(err.Error())
				continue
			}

			slog.Error("unable to establish an ssh connection", "error", err)
			os.Exit(1)
		}

		slog.Info("ssh connection established", "from", sshConn.RemoteAddr(), "clientVersion", sshConn.ClientVersion())

		go ssh.DiscardRequests(reqs)
		go handleChannels(chans)
	}

	// db = setupDatabase(&config)
	// setupServices(&config, db)
	// setupHandlers(&config, db)

	/*
	 * Setup routes
	 */
	// slog.Debug("setting up routes...")

	// routes := []Route{
	// 	{Path: "/", Method: http.MethodGet, Handler: photoHandlers.GetSearchPhotos},
	// 	{Path: "/api/photos", Method: http.MethodGet, Handler: photoHandlers.ApiGetSearchPhotos},
	// }

	// httpServer, quit := setupRouter(&config, routes)
	// slog.Info("server started")

	// <-quit
	// shutdown(httpServer)
}

func handleChannels(chans <-chan ssh.NewChannel) {
	for newChannel := range chans {
		slog.Info("channel type", "type", newChannel.ChannelType())

		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()

		if err != nil {
			slog.Error("could not accept channel", "error", err)
			continue
		}

		slog.Info("channel to accept requests created. creating goroutine to process...")

		go func(in <-chan *ssh.Request) {
			for req := range in {
				slog.Info("received request", "type", req.Type, "payload", req.Payload)
				ok := false

				switch req.Type {
				case "subsystem":
					if string(req.Payload[4:]) == "sftp" {
						ok = true
					}
				}

				req.Reply(ok, nil)
			}
		}(requests)

		slog.Info("setting up server...")

		serverOptions := []sftp.ServerOption{
			sftp.WithDebug(os.Stderr),
		}

		server, err := sftp.NewServer(channel, serverOptions...)

		if err != nil {
			slog.Error("could not start server for session", "error", err)
			os.Exit(1)
		}

		if err := server.Serve(); err != nil {
			if err != io.EOF {
				slog.Error("session ended with an error", "error", err)
			}
		}

		server.Close()
		slog.Info("session ended.")
	}
}

func loadPrivateKey(path string) (ssh.Signer, error) {
	var (
		err     error
		b       []byte
		private ssh.Signer
	)

	if b, err = os.ReadFile(path); err != nil {
		return nil, fmt.Errorf("could not read private server key '%s': %w", path, err)
	}

	if private, err = ssh.ParsePrivateKey(b); err != nil {
		return nil, fmt.Errorf("could not parse private key '%s': %w", path, err)
	}

	return private, nil
}

func loadAuthorizedKeys(path string) (map[string]bool, error) {
	var (
		err error
		b   []byte
	)

	if b, err = os.ReadFile(path); err != nil {
		return nil, fmt.Errorf("could not read authorized keys '%s': %w", path, err)
	}

	authorizedKeys := make(map[string]bool)

	for len(b) > 0 {
		publicKey, _, _, rest, err := ssh.ParseAuthorizedKey(b)

		if err != nil {
			return nil, fmt.Errorf("could not parse authorized key file '%s': %w", path, err)
		}

		authorizedKeys[string(publicKey.Marshal())] = true
		b = rest
	}

	return authorizedKeys, nil
}

func publicKeyAuth(authorizedKeys map[string]bool) func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if authorizedKeys[string(key.Marshal())] {
			return nil, nil
		}

		return nil, fmt.Errorf("user '%q' is not authorized: %w", conn.User(), ErrUnauthorizedUser)
	}
}

func (dr SFTPRequestHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	var (
		attrs *sftp.FileStat
	)

	attrs = r.Attributes()

	slog.Info("writing file", "filepath", r.Filepath, "size", attrs.Size)
	f, _ := os.Open("./test")
	return f, nil
}
