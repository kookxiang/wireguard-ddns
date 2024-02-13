package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.zx2c4.com/wireguard/wgctrl"
	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

var endPoints = make(map[string]string)

func loadConfig(filePath string) error {
	file, openErr := os.OpenFile(filePath, os.O_RDONLY, 0o644)
	if openErr != nil {
		return openErr
	}
	scanner := bufio.NewScanner(file)
	inPeerBlock := false
	var publicKey, endPoint string
	for scanner.Scan() {
		content := scanner.Text()
		if content == "[Peer]" || content == "[WireGuardPeer]" {
			inPeerBlock = true
			publicKey = ""
			endPoint = ""
		} else if !inPeerBlock || len(content) < 2 {
			continue
		} else if content[0] == '[' {
			if publicKey != "" {
				endPoints[publicKey] = endPoint
			}
			inPeerBlock = false
		}
		parts := strings.SplitN(content, "=", 2)
		if len(parts) < 2 {
			continue
		}
		switch strings.TrimSpace(parts[0]) {
		case "PublicKey":
			publicKey = strings.TrimSpace(parts[1])
		case "Endpoint":
			if index := strings.LastIndex(parts[1], ":"); index > 0 {
				endPoint = strings.TrimSpace(parts[1][:index])
			} else {
				endPoint = strings.TrimSpace(parts[1])
			}
		}
	}
	if publicKey != "" {
		endPoints[publicKey] = endPoint
	}
	return nil
}

func loadConfigFromPattern(pattern string) {
	filePaths, _ := filepath.Glob(pattern)
	for _, filePath := range filePaths {
		err := loadConfig(filePath)
		if err != nil {
			fmt.Println("load config", filePath, "failed:", err)
		}
	}
}

func updatePeerEndPoint() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()

	client, err := wgctrl.New()
	if err != nil {
		panic(err)
	}
	defer client.Close()
	wgDevices, err := client.Devices()
	if err != nil {
		panic(err)
	}
	for _, device := range wgDevices {
		newPeerConfig := make([]wgtypes.PeerConfig, 0)
		for _, peer := range device.Peers {
			if time.Now().Sub(peer.LastHandshakeTime) < 3*time.Minute {
				continue
			}
			publicKey := peer.PublicKey.String()
			endPoint, exists := endPoints[publicKey]
			if !exists {
				fmt.Printf("cannot find endpoint config for peer %s on device %s\n", publicKey, device.Name)
				continue
			}
			changed := true
			remoteAddresses, resolveErr := net.LookupIP(endPoint)
			if resolveErr != nil {
				fmt.Printf("failed to resolve %s: %s\n", endPoint, resolveErr)
				continue
			}
			for _, ip := range remoteAddresses {
				if peer.Endpoint.IP.Equal(ip) {
					changed = false
					break
				}
			}
			if !changed {
				continue
			}
			fmt.Printf("peer %s address changed: %s -> %s\n", endPoint, peer.Endpoint.IP, remoteAddresses[0])
			peer.Endpoint.IP = remoteAddresses[0]
			newPeerConfig = append(newPeerConfig, wgtypes.PeerConfig{
				PublicKey:  peer.PublicKey,
				Endpoint:   peer.Endpoint,
				UpdateOnly: true,
			})
		}

		if len(newPeerConfig) > 0 {
			configureErr := client.ConfigureDevice(device.Name, wgtypes.Config{
				Peers: newPeerConfig,
			})
			if configureErr != nil {
				panic(configureErr)
			}
		}
	}
}

func main() {
	loadConfigFromPattern("/etc/wireguard/*.conf")
	loadConfigFromPattern("/etc/systemd/network/*.netdev")

	for key, endPoint := range endPoints {
		fmt.Printf("peer %s -> %s\n", key, endPoint)
	}

	fmt.Println("start peer address update services")

	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ticker.C:
			updatePeerEndPoint()
		}
	}
}
