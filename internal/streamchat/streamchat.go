package streamchat

import (
	"context"
	"time"

	stream "github.com/GetStream/stream-chat-go/v6"
)

// Client wraps the official Stream Chat Go SDK, mirroring the two calls
// auth.controller.ts makes: upsertUser (on register/createDoctor) and
// createToken (on login/loginDoctor).
type Client struct {
	sc *stream.Client
}

func New(apiKey, apiSecret string) (*Client, error) {
	sc, err := stream.NewClient(apiKey, apiSecret)
	if err != nil {
		return nil, err
	}
	return &Client{sc: sc}, nil
}

type UpsertUserInput struct {
	ID    string
	Name  string
	Email string
	Role  string // optional, only set for doctors
}

func (c *Client) UpsertUser(in UpsertUserInput) error {
	user := &stream.User{
		ID:        in.ID,
		Name:      in.Name,
		ExtraData: map[string]interface{}{"email": in.Email},
	}
	if in.Role != "" {
		user.Role = in.Role
	}
	_, err := c.sc.UpsertUser(context.Background(), user)
	return err
}

// CreateToken mirrors streamClient.createToken(userId) from the Node SDK,
// which issues a non-expiring token.
func (c *Client) CreateToken(userID string) (string, error) {
	return c.sc.CreateToken(userID, time.Time{})
}
