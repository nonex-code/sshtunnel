package main

import (
	"fmt"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"os"
	"sshtunnel/core"
	"strings"
)

// 初始化root命令
func init() {
	rootCmd.AddCommand(tunnelWithKeyCmd)
	rootCmd.AddCommand(tunnelWithPassCmd)

	// 设置tunnelWithKeyCmd的标志
	tunnelWithKeyCmd.Flags().StringP("ssh", "s", "", "SSH服务地址：user@host:port")
	tunnelWithKeyCmd.Flags().StringP("remote", "r", "", "远程服务器监听地址：host:port")
	tunnelWithKeyCmd.Flags().StringP("local", "l", "", "本地服务地址：host:port")
	tunnelWithKeyCmd.Flags().StringP("key", "k", "", "密钥路径")

	// 设置tunnelWithPassCmd的标志
	tunnelWithPassCmd.Flags().StringP("ssh", "s", "", "SSH服务地址：user@host:port")
	tunnelWithPassCmd.Flags().StringP("remote", "r", "", "远程服务器监听地址：host:port")
	tunnelWithPassCmd.Flags().StringP("local", "l", "", "本地服务地址：host:port")
	tunnelWithPassCmd.Flags().StringP("passwd", "p", "", "SSH服务密码")
}

// 定义root命令
var rootCmd = &cobra.Command{
	Use:   "ssh-tunnel",
	Short: "SSH隧道创建工具",
	Long:  `ssh-tunnel 是一个用于创建SSH隧道的命令行工具，支持使用密钥或密码进行认证`,
}

// 使用密钥创建SSH隧道的命令
var tunnelWithKeyCmd = &cobra.Command{
	Use:   "tunnel-key",
	Short: "使用密钥创建SSH隧道",
	Long:  `使用密钥进行SSH认证并创建隧道`,
	Run: func(cmd *cobra.Command, args []string) {
		executeTunnelWithKey(cmd)
	},
}

// 使用密码创建SSH隧道的命令
var tunnelWithPassCmd = &cobra.Command{
	Use:   "tunnel-pass",
	Short: "使用密码创建SSH隧道",
	Long:  `使用SSH密码进行认证并创建隧道`,
	Run: func(cmd *cobra.Command, args []string) {
		executeTunnelWithPass(cmd)
	},
}

// 执行使用密钥创建隧道的逻辑
func executeTunnelWithKey(cmd *cobra.Command) {
	// 获取命令行参数
	sshAddr, _ := cmd.Flags().GetString("ssh")
	remoteAddr, _ := cmd.Flags().GetString("remote")
	localAddr, _ := cmd.Flags().GetString("local")
	keyPath, _ := cmd.Flags().GetString("key")

	// 验证输入参数
	if err := validateInputs(sshAddr, remoteAddr, localAddr, keyPath, ""); err != nil {
		fmt.Println(err)
		return
	}

	// 解析SSH地址
	user, addr, err := parseSSHAddress(sshAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 读取密钥文件
	key, err := os.ReadFile(keyPath)
	if err != nil {
		fmt.Printf("密钥读取失败: %v\n", err)
		return
	}

	// 调用创建SSH隧道的函数
	fmt.Printf("正在使用密钥创建SSH隧道，用户: %s，SSH地址: %s\n", user, addr)
	createSSHTunnel(user, string(key), "", addr, remoteAddr, localAddr)
}

// 执行使用密码创建隧道的逻辑
func executeTunnelWithPass(cmd *cobra.Command) {
	// 获取命令行参数
	sshAddr, _ := cmd.Flags().GetString("ssh")
	remoteAddr, _ := cmd.Flags().GetString("remote")
	localAddr, _ := cmd.Flags().GetString("local")
	passwd, _ := cmd.Flags().GetString("passwd")

	// 验证输入参数
	if err := validateInputs(sshAddr, remoteAddr, localAddr, "", passwd); err != nil {
		fmt.Println(err)
		return
	}

	// 解析SSH地址
	user, addr, err := parseSSHAddress(sshAddr)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 调用创建SSH隧道的函数
	fmt.Printf("正在使用密码创建SSH隧道，用户: %s，SSH地址: %s\n", user, addr)
	createSSHTunnel(user, "", passwd, addr, remoteAddr, localAddr)
}

// 解析SSH服务地址 (user@host:port)
func parseSSHAddress(sshAddr string) (string, string, error) {
	parts := strings.Split(sshAddr, "@")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("SSH服务地址格式错误，应为 user@host:port")
	}
	return parts[0], parts[1], nil
}

// 验证输入参数
func validateInputs(sshAddr, remoteAddr, localAddr, keyPath, passwd string) error {
	if sshAddr == "" {
		return fmt.Errorf("未指定SSH服务地址")
	}
	if remoteAddr == "" {
		return fmt.Errorf("未指定远程服务器监听地址")
	}
	if localAddr == "" {
		return fmt.Errorf("未指定本地服务地址")
	}
	if keyPath == "" && passwd == "" {
		return fmt.Errorf("未指定密钥路径或SSH密码")
	}
	return nil
}

// 主函数，执行root命令
func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// 创建SSH隧道
func createSSHTunnel(user, sshKey, sshPassword, sshServerAddr, remoteListenAddr, localServiceAddr string) {
	var authMethods []ssh.AuthMethod

	if sshKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(sshKey))
		if err != nil {
			fmt.Printf("密钥读取失败: %v\n", err)
			return
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if sshPassword != "" {
		authMethods = append(authMethods, ssh.Password(sshPassword))
	}

	config := ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境中使用安全的 HostKeyCallback
	}

	tunnelConfig := core.SSHTunnelConfig{
		SSHClientConfig:     config,
		SSHServerAddress:    sshServerAddr,
		RemoteListenAddress: remoteListenAddr,
		LocalServiceAddress: localServiceAddr,
	}

	tunnelConfig.ForwardLocalPort()
}
