// Package skills provides IPFS skill implementation using official go-ipfs-http-client
package skills

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
	"time"

	ipfsapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/boxo/files"
	ipath "github.com/ipfs/boxo/coreiface/path"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
)

// IPFSSkill returns the IPFS skill definition
func IPFSSkill() Skill {
	return Skill{
		Name:        "ipfs",
		Description: "IPFS decentralized storage operations: upload content, download by CID, manage pins. Supports Infura, Pinata, and custom IPFS nodes.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"operation": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"upload", "download", "pin", "unpin", "list_pins"},
					"description": "IPFS operation to perform",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to upload (for upload operation)",
				},
				"cid": map[string]interface{}{
					"type":        "string",
					"description": "Content identifier (for download, pin, unpin operations)",
				},
			},
			"required": []string{"operation"},
		},
	}
}

// IPFSHandler handles IPFS skill execution
func IPFSHandler(args map[string]interface{}, configLoader ConfigLoader) (interface{}, error) {
	// Get configuration
	config, err := configLoader.GetIPFSConfig()
	if err != nil {
		return formatErrorResponse(
			"config_error",
			"IPFS configuration is required",
			"Configure IPFS by setting CELESTE_IPFS_API_KEY environment variable or adding to skills.json",
			map[string]interface{}{
				"skill":          "ipfs",
				"config_command": "Set CELESTE_IPFS_API_KEY=<your_key>",
			},
		), nil
	}

	// Get operation
	operation, ok := args["operation"].(string)
	if !ok || operation == "" {
		return formatErrorResponse(
			"validation_error",
			"Operation is required",
			"Specify one of: upload, download, pin, unpin, list_pins",
			map[string]interface{}{
				"skill": "ipfs",
				"field": "operation",
			},
		), nil
	}

	// Create IPFS client
	client, err := createIPFSClient(config)
	if err != nil {
		return formatErrorResponse(
			"connection_error",
			fmt.Sprintf("Failed to connect to IPFS: %v", err),
			"Check your IPFS configuration and network connection",
			map[string]interface{}{
				"skill":    "ipfs",
				"provider": config.Provider,
			},
		), nil
	}

	// Route to appropriate handler
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.TimeoutSeconds)*time.Second)
	defer cancel()

	switch operation {
	case "upload":
		return handleIPFSUpload(ctx, client, args, config)
	case "download":
		return handleIPFSDownload(ctx, client, args)
	case "pin":
		return handleIPFSPin(ctx, client, args)
	case "unpin":
		return handleIPFSUnpin(ctx, client, args)
	case "list_pins":
		return handleIPFSListPins(ctx, client)
	default:
		return formatErrorResponse(
			"validation_error",
			fmt.Sprintf("Unknown operation: %s", operation),
			"Valid operations: upload, download, pin, unpin, list_pins",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": operation,
			},
		), nil
	}
}

// createIPFSClient creates an IPFS HTTP API client with authentication
func createIPFSClient(config IPFSConfig) (*ipfsapi.HttpApi, error) {
	// Determine endpoint based on provider
	endpoint := config.GatewayURL
	if endpoint == "" {
		switch config.Provider {
		case "infura":
			endpoint = "/dns/ipfs.infura.io/tcp/5001/https"
		case "pinata":
			endpoint = "/dns/api.pinata.cloud/tcp/443/https"
		default:
			endpoint = "/ip4/127.0.0.1/tcp/5001" // Local IPFS node
		}
	}

	// Parse multiaddr
	addr, err := multiaddr.NewMultiaddr(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid IPFS endpoint: %w", err)
	}

	// Create HTTP API client
	client, err := ipfsapi.NewApi(addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create IPFS client: %w", err)
	}

	// Set authentication for Infura
	if config.Provider == "infura" && config.ProjectID != "" && config.APISecret != "" {
		auth := base64.StdEncoding.EncodeToString(
			[]byte(fmt.Sprintf("%s:%s", config.ProjectID, config.APISecret)),
		)
		client.Headers.Add("Authorization", "Basic "+auth)
	}

	// Set API key for Pinata
	if config.Provider == "pinata" && config.APIKey != "" {
		client.Headers.Add("pinata_api_key", config.APIKey)
		if config.APISecret != "" {
			client.Headers.Add("pinata_secret_api_key", config.APISecret)
		}
	}

	return client, nil
}

// handleIPFSUpload uploads content to IPFS
func handleIPFSUpload(ctx context.Context, client *ipfsapi.HttpApi, args map[string]interface{}, config IPFSConfig) (interface{}, error) {
	// Get content
	content, ok := args["content"].(string)
	if !ok || content == "" {
		return formatErrorResponse(
			"validation_error",
			"Content is required for upload operation",
			"Provide content to upload to IPFS",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "upload",
			},
		), nil
	}

	// Upload content - wrap in files.Node
	reader := strings.NewReader(content)
	fileNode := files.NewReaderFile(reader)
	path, err := client.Unixfs().Add(ctx, fileNode)
	if err != nil {
		return formatErrorResponse(
			"upload_error",
			fmt.Sprintf("Failed to upload to IPFS: %v", err),
			"Check your IPFS configuration and try again",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "upload",
			},
		), nil
	}

	// Build gateway URL
	gatewayURL := ""
	if config.GatewayURL != "" {
		gatewayURL = fmt.Sprintf("%s/ipfs/%s", config.GatewayURL, path.Cid().String())
	} else {
		gatewayURL = fmt.Sprintf("https://ipfs.io/ipfs/%s", path.Cid().String())
	}

	return map[string]interface{}{
		"success":     true,
		"cid":         path.Cid().String(),
		"size":        len(content),
		"gateway_url": gatewayURL,
		"message":     "Content successfully uploaded to IPFS",
	}, nil
}

// handleIPFSDownload downloads content from IPFS by CID
func handleIPFSDownload(ctx context.Context, client *ipfsapi.HttpApi, args map[string]interface{}) (interface{}, error) {
	// Get CID
	cidStr, ok := args["cid"].(string)
	if !ok || cidStr == "" {
		return formatErrorResponse(
			"validation_error",
			"CID is required for download operation",
			"Provide a valid IPFS Content Identifier (CID)",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "download",
			},
		), nil
	}

	// Parse CID
	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return formatErrorResponse(
			"validation_error",
			fmt.Sprintf("Invalid CID: %v", err),
			"Provide a valid IPFS Content Identifier",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "download",
				"cid":       cidStr,
			},
		), nil
	}

	// Download content
	path := ipath.New("/ipfs/" + parsedCID.String())
	node, err := client.Unixfs().Get(ctx, path)
	if err != nil {
		return formatErrorResponse(
			"download_error",
			fmt.Sprintf("Failed to download from IPFS: %v", err),
			"Check that the CID exists and is accessible",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "download",
				"cid":       cidStr,
			},
		), nil
	}
	defer node.Close()

	// Read content from file node
	fileNode := files.ToFile(node)
	if fileNode == nil {
		return formatErrorResponse(
			"download_error",
			"Content is not a file",
			"The CID may point to a directory",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "download",
				"cid":       cidStr,
			},
		), nil
	}
	content, err := io.ReadAll(fileNode)
	if err != nil {
		return formatErrorResponse(
			"download_error",
			fmt.Sprintf("Failed to read content: %v", err),
			"",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "download",
			},
		), nil
	}

	return map[string]interface{}{
		"success": true,
		"cid":     cidStr,
		"content": string(content),
		"size":    len(content),
		"message": "Content successfully downloaded from IPFS",
	}, nil
}

// handleIPFSPin pins content on IPFS
func handleIPFSPin(ctx context.Context, client *ipfsapi.HttpApi, args map[string]interface{}) (interface{}, error) {
	// Get CID
	cidStr, ok := args["cid"].(string)
	if !ok || cidStr == "" {
		return formatErrorResponse(
			"validation_error",
			"CID is required for pin operation",
			"Provide a valid IPFS Content Identifier to pin",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "pin",
			},
		), nil
	}

	// Parse CID
	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return formatErrorResponse(
			"validation_error",
			fmt.Sprintf("Invalid CID: %v", err),
			"Provide a valid IPFS Content Identifier",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "pin",
				"cid":       cidStr,
			},
		), nil
	}

	// Pin content
	path := ipath.New("/ipfs/" + parsedCID.String())
	err = client.Pin().Add(ctx, path)
	if err != nil {
		return formatErrorResponse(
			"pin_error",
			fmt.Sprintf("Failed to pin content: %v", err),
			"Check that the CID exists and is accessible",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "pin",
				"cid":       cidStr,
			},
		), nil
	}

	return map[string]interface{}{
		"success": true,
		"cid":     cidStr,
		"message": "Content successfully pinned on IPFS",
	}, nil
}

// handleIPFSUnpin unpins content from IPFS
func handleIPFSUnpin(ctx context.Context, client *ipfsapi.HttpApi, args map[string]interface{}) (interface{}, error) {
	// Get CID
	cidStr, ok := args["cid"].(string)
	if !ok || cidStr == "" {
		return formatErrorResponse(
			"validation_error",
			"CID is required for unpin operation",
			"Provide a valid IPFS Content Identifier to unpin",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "unpin",
			},
		), nil
	}

	// Parse CID
	parsedCID, err := cid.Decode(cidStr)
	if err != nil {
		return formatErrorResponse(
			"validation_error",
			fmt.Sprintf("Invalid CID: %v", err),
			"Provide a valid IPFS Content Identifier",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "unpin",
				"cid":       cidStr,
			},
		), nil
	}

	// Unpin content
	path := ipath.New("/ipfs/" + parsedCID.String())
	err = client.Pin().Rm(ctx, path)
	if err != nil {
		return formatErrorResponse(
			"unpin_error",
			fmt.Sprintf("Failed to unpin content: %v", err),
			"Check that the content is currently pinned",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "unpin",
				"cid":       cidStr,
			},
		), nil
	}

	return map[string]interface{}{
		"success": true,
		"cid":     cidStr,
		"message": "Content successfully unpinned from IPFS",
	}, nil
}

// handleIPFSListPins lists all pinned content
func handleIPFSListPins(ctx context.Context, client *ipfsapi.HttpApi) (interface{}, error) {
	// List pins
	pins, err := client.Pin().Ls(ctx)
	if err != nil {
		return formatErrorResponse(
			"list_error",
			fmt.Sprintf("Failed to list pins: %v", err),
			"",
			map[string]interface{}{
				"skill":     "ipfs",
				"operation": "list_pins",
			},
		), nil
	}

	// Convert to string array
	var cidList []string
	for pin := range pins {
		if pin.Err() != nil {
			continue
		}
		cidList = append(cidList, pin.Path().Cid().String())
	}

	return map[string]interface{}{
		"success": true,
		"pins":    cidList,
		"count":   len(cidList),
		"message": fmt.Sprintf("Found %d pinned items", len(cidList)),
	}, nil
}
