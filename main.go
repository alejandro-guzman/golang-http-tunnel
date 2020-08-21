package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/net/proxy"
)

type httpTunnel struct {
	proxyHost string
	forward   proxy.Dialer
}

func (t *httpTunnel) Dial(network, addr string) (c net.Conn, err error) {
	// connect to proxy
	conn, err := t.forward.Dial("tcp", t.proxyHost)
	if err != nil {
		return nil, err
	}

	// prepare addr as URL
	addrURL, err := url.Parse(addr)

	// make http request
	request := &http.Request{
		Method: "CONNECT",
		URL:    addrURL,
		Host:   addrURL.Host,
	}

	// nc proxy 3128
	// CONNECT dest port HTTP/1.1

	fmt.Printf("request: %+v\n", request)

	// send request through connection
	if err := request.Write(conn); err != nil {
		return nil, err
	}

	// check response for errors
	response, err := http.ReadResponse(bufio.NewReader(conn), request)
	if err != nil {
		return nil, err
	}

	fmt.Printf("response: %+v\n", response)

	if response.StatusCode != 200 {
		return nil, fmt.Errorf("Response: %s", response.Status)
	}

	// return connection
	return conn, nil
}

func getHTTPTunnel(u *url.URL, f proxy.Dialer) (proxy.Dialer, error) {
	httpTunnel := &httpTunnel{
		proxyHost: u.Host,
		forward:   f,
	}
	return httpTunnel, nil
}

func main() {

	proxyHost := flag.String("proxyhost", "localhost", "Proxy host to tunnel through")
	proxyPort := flag.Int("proxyport", 3128, "Proxy port")
	addrHost := flag.String("addrhost", "", "Remote SSH server host/ip")
	addrPort := flag.Int("addrport", 22, "Remote SSH server port")

	flag.Parse()

	proxyURLStr := fmt.Sprintf("http://%s:%d", *proxyHost, *proxyPort)
	destinationURLStr := fmt.Sprintf("//%s:%d", *addrHost, *addrPort)

	usr, _ := user.Current()
	dir := usr.HomeDir
	privateKey := filepath.Join(dir, ".ssh", "id_rsa")
	// fmt.Printf("SSH key: %s\n", privateKey)

	// register the custom http tunnel dialer
	proxy.RegisterDialerType("http", getHTTPTunnel)

	// prepare the proxy url
	proxyURL, err := url.Parse(proxyURLStr)
	if err != nil {
		panic(fmt.Errorf("Error encountered when parsing URL [%s]: [%s]", proxyURLStr, err))
	}

	// create custom http tunnel dialer (calls getHTTPTunbel)
	httpTunnelDialer, err := proxy.FromURL(proxyURL, proxy.Direct)
	if err != nil {
		panic(fmt.Errorf("Error encountered when creating custom http tunnel dialer: [%s]", err))
	}

	// connect to destination using custom http tunnel dialer
	conn, err := httpTunnelDialer.Dial("tcp", destinationURLStr)
	if err != nil {
		panic(fmt.Errorf("Error encountered when dialing [%s]: [%s]", destinationURLStr, err))
	}

	// private key
	key, err := ioutil.ReadFile(privateKey)
	if err != nil {
		panic(fmt.Errorf("Error reading private key file: [%s]", err))
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		panic(fmt.Errorf("Error parsing private key: [%s]", err))
	}

	// sample ssh client config
	config := &ssh.ClientConfig{
		User: "root", // usr.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
			// ssh.Password("password"),
		},
		HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			return nil
		},
	}

	fmt.Printf("SSH config: %+v\n", config)

	// create ssh client
	c, chans, reqs, err := ssh.NewClientConn(conn, destinationURLStr, config)
	if err != nil {
		panic(fmt.Errorf("Error encountered when creating ssh client conn: [%s]", err))
	}

	client := ssh.NewClient(c, chans, reqs)
	session, err := client.NewSession()
	if err != nil {
		panic(fmt.Errorf("Error creating session: [%s]", err))
	}
	defer client.Close()

	// run command on remote
	out, err := session.CombinedOutput("ls -la")
	if err != nil {
		panic(fmt.Errorf("Error running command: [%s]", err))
	}

	// print command output
	fmt.Println(string(out))

	// create sftp client (wrapper around SSH client)
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		panic(fmt.Errorf("Error encountered when creating sftp client conn: [%s]", err))
	}
	defer sftpClient.Close()

	filename := "./copyme.txt"

	// create destination file
	dstFile, err := sftpClient.Create(filename)
	if err != nil {
		panic(fmt.Errorf("Error creating dest file: [%s]", err))
	}
	defer dstFile.Close()

	// create source file
	srcFile, err := os.Open(filename)
	if err != nil {
		panic(fmt.Errorf("Error creating source file: [%s]", err))
	}

	// copy source file to destination file
	bytes, err := io.Copy(dstFile, srcFile)
	if err != nil {
		panic(fmt.Errorf("Error copying file: [%s]", err))
	}
	fmt.Printf("%d bytes copied\n", bytes)

	session, err = client.NewSession()
	if err != nil {
		panic(fmt.Errorf("Error creating session: [%s]", err))
	}

	out, err = session.CombinedOutput("cat " + filename)
	if err != nil {
		panic(fmt.Errorf("Error running command: [%s]", err))
	}

	// print command output
	fmt.Println(string(out))

}
