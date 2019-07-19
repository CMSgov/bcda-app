package client

// SSASClient is a client for interacting with the System-to-System Authentication Service.
type SSASClient struct{}

// NewSSASClient creates and returns an SSASClient.
func NewSSASClient() (*SSASClient, error) {
	return &SSASClient{}, nil
}

// CreateSystem POSTs to the SSAS /system endpoint to create a system.
func (c *SSASClient) CreateSystem() ([]byte, error) {
	return nil, nil
}

// GetPublicKey GETs the SSAS /system/{systemID}/key endpoint to retrieve a system's public key.
func (c *SSASClient) GetPublicKey(systemID int) ([]byte, error) {
	return nil, nil
}

// ResetCredentials PUTs to the SSAS /system/{systemID}/credentials endpoint to reset the system's credentials.
func (c *SSASClient) ResetCredentials(systemID int) ([]byte, error) {
	return nil, nil
}

// DeleteCredentials DELETEs from the SSAS /system/{systemID}/credentials endpoint to deactivate credentials associated with the system.
func (c *SSASClient) DeleteCredentials(systemID int) ([]byte, error) {
	return nil, nil
}
