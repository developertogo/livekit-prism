package main

import (
	"context"
	"fmt"
	"log"

	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// LiveKitClient manages connection-pooled gRPC interactions with the LiveKit SFU.
type LiveKitClient struct {
	roomClient *lksdk.RoomServiceClient
	url        string
}

// NewLiveKitClient initializes a RoomServiceClient wrapper.
func NewLiveKitClient(url, apiKey, apiSecret string) *LiveKitClient {
	roomClient := lksdk.NewRoomServiceClient(url, apiKey, apiSecret)
	return &LiveKitClient{
		roomClient: roomClient,
		url:        url,
	}
}

// SendVisualCard broadcasts a JSON payload to a LiveKit room over the WebRTC Data Channel.
func (c *LiveKitClient) SendVisualCard(ctx context.Context, roomName string, topic string, payload []byte) error {
	if c.roomClient == nil {
		return fmt.Errorf("room client is not initialized")
	}

	req := &livekit.SendDataRequest{
		Room:  roomName,
		Data:  payload,
		Kind:  livekit.DataPacket_RELIABLE,
		Topic: &topic,
	}

	res, err := c.roomClient.SendData(ctx, req)
	if err != nil {
		log.Printf("[LiveKitClient] Failed to send visual card to room %s: %v", roomName, err)
		return err
	}

	log.Printf("[LiveKitClient] Successfully sent visual card (%d bytes) to room %s, response: %v", len(payload), roomName, res)
	return nil
}
