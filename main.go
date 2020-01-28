// Copyright 2018 The goftp Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// This is a very simple ftpd server using this library as an example
// and as something to run tests against.
package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"

	azblob "github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/goftp/server"
	filedriver "github.com/tryy3/ftp-to-blob/ftpazuredriver"
)

func main() {
	var (
		username      = flag.String("username", "admin", "Username for login")
		password      = flag.String("password", "password", "Password for login")
		port          = flag.Int("port", 2121, "Port")
		hostname      = flag.String("hostname", "localhost", "Hostname")
		publicIP      = flag.String("public-ip", "", "Public IP")
		passivePorts  = flag.String("passive-ports", "", "Passive Ports")
		TLS           = flag.Bool("tls", false, "TLS")
		certFile      = flag.String("cert-file", "", "Certificate File")
		keyFile       = flag.String("key-file", "", "Key File")
		explicitFTPS  = flag.Bool("ftps", false, "Explicit FTPS")
		welcome       = flag.String("welcome", "", "Welcome message")
		accountName   = flag.String("account-name", "", "Azure Blob Account")
		accountKey    = flag.String("account-key", "", "Azure Blob key")
		containerName = flag.String("container-name", "", "Azure container name")
	)
	flag.Parse()
	fmt.Printf("{\n\tusername:%s\n\tpassword:%shostname:%s\n\tport:%d\n\tpublicIP:%s\n\tpassivePorts:%s\n\tTLS:%t\n\tcertFile:%s\n\tkeyFile:%s\n\texplicitFTPS:%t\n\twelcome:%s\n}\n",
		*username,
		*password,
		*hostname,
		*port,
		*publicIP,
		*passivePorts,
		*TLS,
		*certFile,
		*keyFile,
		*explicitFTPS,
		*welcome,
	)

	// Use your Storage account's name and key to create a credential object; this is required to sign a SAS.
	credential, err := azblob.NewSharedKeyCredential(*accountName, *accountKey)
	if err != nil {
		log.Fatal(err)
	}
	p := azblob.NewPipeline(credential, azblob.PipelineOptions{})
	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", *accountName, *containerName))

	container := azblob.NewContainerURL(*URL, p)

	factory := &filedriver.FileDriverFactory{
		AccountName:   *accountName,
		AccountKey:    *accountKey,
		ContainerName: *containerName,
		Container:     container,
	}

	opts := &server.ServerOpts{
		Factory:        factory,
		Auth:           &server.SimpleAuth{Name: *username, Password: *password},
		Hostname:       *hostname,
		PublicIp:       *publicIP,
		PassivePorts:   *passivePorts,
		Port:           *port,
		TLS:            *TLS,
		CertFile:       *certFile,
		KeyFile:        *keyFile,
		ExplicitFTPS:   *explicitFTPS,
		WelcomeMessage: *welcome,
	}

	log.Printf("Starting ftp server on %v:%v", opts.Hostname, opts.Port)
	log.Printf("Username %v, Password %v", *username, *password)
	server := server.NewServer(opts)
	err = server.ListenAndServe()
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
