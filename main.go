package main

import (
	"flag"
	"fmt"
	"golang.org/x/crypto/ssh"
	"os"
	"sshtunnel/core"
	"strings"
)

func main() {
	var sshServerAddr string
	var remoteLicAddr string
	var localServerAddr string
	var PemPath string
	var sshPasswd string
	flag.StringVar(&sshServerAddr, "s", "", "ssh服务地址：user@host:port")
	flag.StringVar(&remoteLicAddr, "r", "", "远程服务器监听地址：host:port")
	flag.StringVar(&localServerAddr, "l", "", "本地服务地址：host:port")
	flag.StringVar(&PemPath, "i", "", "pem证书路径")
	flag.StringVar(&sshPasswd, "p", "", "ssh服务密码")
	flag.Parse()
	strs := strings.Split(sshServerAddr, "@")
	user := strs[0]
	sshAddr := strs[1]
	if PemPath == "" && sshPasswd == "" {
		fmt.Println("未指定证书或密码")
		return
	}
	if PemPath != "" || sshPasswd != "" {
		if PemPath != "" {
			key, err := os.ReadFile(PemPath)
			if err != nil {
				fmt.Printf("秘钥读取失败: %v", err)
				return
			}
			createSSHTunnelWithPEM(user, string(key), sshAddr, remoteLicAddr, localServerAddr)
		}
		if sshPasswd != "" {
			createSSHTunnelWithPassword(user, sshPasswd, sshAddr, remoteLicAddr, localServerAddr)
		}

	}
	if PemPath != "" && sshPasswd != "" {
		key, err := os.ReadFile(PemPath)
		if err != nil {
			fmt.Printf("秘钥读取失败file: %v", err)
			return
		}
		createSSHTunnelWithPEM(user, string(key), sshAddr, remoteLicAddr, localServerAddr)
	}
}

func createSSHTunnelWithPEM(user string, sshKey string, remoteServerAddr string, remoteLicAddr string, localServerAddr string) {
	signer, err := ssh.ParsePrivateKey([]byte(sshKey))
	if err != nil {
		fmt.Printf("秘钥读取失败key: %v", err)
		return
	}
	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中使用安全的 HostKeyCallback
	}
	tunnelConf := core.TunnelConf{
		Config:           config,
		RemoteServerAddr: remoteServerAddr,
		RemoteLicAddr:    remoteLicAddr,
		LocalServerAddr:  localServerAddr,
	}
	tunnelConf.ConnectAndForward()
}

func createSSHTunnelWithPassword(user string, sshPassword string, remoteServerAddr string, remoteLicAddr string, localServerAddr string) {
	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中使用安全的 HostKeyCallback
	}
	tunnelConf := core.TunnelConf{
		Config:           config,
		RemoteServerAddr: remoteServerAddr,
		RemoteLicAddr:    remoteLicAddr,
		LocalServerAddr:  localServerAddr,
	}
	tunnelConf.ConnectAndForward()
}
