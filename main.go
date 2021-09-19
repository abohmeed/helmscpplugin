package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

type Action int

const (
	Upload = iota
	Download
)
const Protocol = "scp"

var key, chartPath string
var action Action
var helmBin string

type URL struct {
	username string
	host     string
	port     string
	path     string
}

// helm scp push /path/to/chart scp://ahmed@192.168.2.164:22/myrepo
func detokenize(url string) (URL, error) {
	regex := `scp:\/\/(\w+)@(\d+\.\d+\.\d+\.\d+):?(\d+)?(.*)$`
	r := regexp.MustCompile(regex)
	if !r.MatchString(url) {
		return URL{}, errors.New("INVALID SCP URL")
	}
	m := r.FindAllStringSubmatch(url, -1)
	username := m[0][1]
	host := m[0][2]
	port := "22"
	if m[0][3] != "" {
		port = m[0][3]
	}
	remotePath := "/home/" + username + "/"
	if m[0][4] != "" {
		remotePath = m[0][4]
	}
	return URL{
		username: username,
		host:     host,
		port:     port,
		path:     remotePath,
	}, nil
}
func initialize() (URL, error) {
	var url URL
	var err error
	if os.Getenv("SCP_KEY") != "" {
		key = os.Getenv("SCP_KEY")
	}
	if len(os.Args) == 4 {
		if os.Args[1] == "push" {
			url, err = detokenize(os.Args[3])
			if err != nil {
				return url, errors.New("please make sure the URL is scp://username@host[:port]/path")
			}
			chartPath = os.Args[2]
			action = Upload
		}
	} else if len(os.Args) == 5 {
		url, err = detokenize(os.Args[4])
		if err != nil {
			return URL{}, errors.New("please make sure the URL is scp://username@host[:port]/path")
		}
		action = Download
	} else {
		return URL{}, errors.New("incorrect arguments.\nUsage:\nhelmscp push /path/to/chart scp://username@hostname[:port]/path/to/remote\nOR\nhelmscp scp://username@hostname:port/path/to/chart")
	}
	return url, nil
}
func main() {
	url, err := initialize()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if action == Upload {
		chartFile, err := Package(chartPath)
		if err != nil {
			log.Fatalf("Error while packaging the chart: %s", err)
			return
		}
		err = Scp(chartFile, url, Upload)
		if err != nil {
			log.Fatalf("Error while uploading the archive: %s", err)
			return
		}
		fmt.Printf("Success!\n")
	} else {
		err = Scp("", url, Download)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}
func Package(chartPath string) (string, error) {
	if os.Getenv("HELM_BIN") != "" {
		helmBin = os.Getenv("HELM_BIN")
	} else {
		helmBin = "helm"
	}
	fmt.Printf("Packaging chart from %s\n", chartPath)
	cmd := exec.Command(helmBin, "package", chartPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	chartNameFullPath := strings.Split(out.String(), ":")[1]
	chartNameFullPath = strings.Trim(chartNameFullPath, "\n")
	chartNameFullPath = strings.Trim(chartNameFullPath, " ")
	return chartNameFullPath, nil
}
func Scp(filename string, url URL, action Action) error {
	clientConfig, _ := auth.PrivateKey(url.username, key, ssh.InsecureIgnoreHostKey())
	client := scp.NewClient(url.host+":"+url.port, &clientConfig)
	err := client.Connect()
	if err != nil {
		log.Fatal("Couldn't establish a connection to the remote server ", err)
		return err
	}
	// Close client connection after the file has been copied
	defer client.Close()
	baseFileName := filepath.Base(filename)
	if action == Upload {
		if url.path[len(url.path)-1:] != "/" {
			url.path = url.path + "/"
		}
		// Open a file
		f, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Could not open %s: %s", filename, err)
			return err
		}
		defer f.Close()
		defer os.Remove(filename)
		fmt.Printf("Uploading %s to %s at %s@%s:%s\n", baseFileName, url.path, url.username, url.host, url.port)
		// Finaly, copy the file over
		// Usage: CopyFile(fileReader, remotePath, permission)
		err = client.CopyFile(f, url.path+baseFileName, "0644")
		if err != nil {
			return err
		}
		fmt.Printf("Cleaning up\n")
		return nil
	} else {
		remoteFile := url.path
		// Must point to a file not a directory
		if strings.HasSuffix(remoteFile, "/") {
			return errors.New("remote path must be a file not a directory")
		}
		sshClient, err := ssh.Dial("tcp", url.host+":"+url.port, &clientConfig)
		if err != nil {
			return err
		}
		defer sshClient.Close()
		session, err := sshClient.NewSession()
		if err != nil {
			return err
		}
		defer session.Close()
		if err := session.Run("stat " + remoteFile); err != nil {
			return fmt.Errorf("could not download %s", remoteFile)
		}
		err = client.CopyFromRemote(os.Stdout, remoteFile)
		if err != nil {
			return err
		}
		return nil
	}
}
