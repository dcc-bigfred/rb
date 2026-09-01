package decoder

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"
)

const DefaultHTTPAddress = "http://192.168.4.1"

const (
	soundPackageClearEndpoint      = "/delete?p=/%d/all"
	soundPackageDeleteFileEndpoint = "/delete?p=/%d/%s"
	soundPackageListEndpoint       = "/?p=/%d/"
	soundPackageUploadEndpoint     = "/upload?p=/%d/%s"
	defaultTimeout                 = 10 * time.Second
)

// Option configures a Client.
type Option func(*Client)

// WithTimeout sets the HTTP client timeout in seconds.
func WithTimeout(seconds uint16) Option {
	return func(d *Client) {
		d.client.Timeout = time.Duration(seconds) * time.Second
	}
}

// WithBaseURL overrides the decoder HTTP base URL (default http://192.168.4.1).
func WithBaseURL(base string) Option {
	return func(d *Client) {
		d.baseURL = base
	}
}

// Client talks to a RailBOX RB23xx decoder over its Soft-AP HTTP interface.
type Client struct {
	client  *http.Client
	baseURL string
}

// NewClient returns an HTTP client for the decoder's sound-slot API.
func NewClient(opts ...Option) *Client {
	d := &Client{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: DefaultHTTPAddress,
	}
	for _, opt := range opts {
		opt(d)
	}
	return d
}

func (d *Client) httpGet(endpoint string) (*http.Response, error) {
	resp, err := d.client.Get(d.baseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to loco wifi (are you connected to loco wifi? is loco wifi function on?): %w", err)
	}
	return resp, nil
}

// ClearSoundSlot removes all sound files from the given slot on the decoder.
func (d *Client) ClearSoundSlot(slot uint8) error {
	resp, err := d.httpGet(fmt.Sprintf(soundPackageClearEndpoint, slot))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// reFileEntry matches a file row in the listing HTML, capturing name and size in KB.
// Example row fragment: placeholder='F11_Decouple.wav'> </td><td>file</td><td align='right'>108</td>
var reFileEntry = regexp.MustCompile(`placeholder='([^']+)'[^<]*</td><td>file</td><td[^>]*>(\d+)</td>`)

// RemoteFileInfo holds metadata about a file on the decoder.
type RemoteFileInfo struct {
	Name   string
	SizeKB int64
}

func parseSoundListing(body []byte) []RemoteFileInfo {
	matches := reFileEntry.FindAllSubmatch(body, -1)
	files := make([]RemoteFileInfo, 0, len(matches))
	for _, m := range matches {
		var sizeKB int64
		fmt.Sscan(string(m[2]), &sizeKB)
		files = append(files, RemoteFileInfo{
			Name:   string(m[1]),
			SizeKB: sizeKB,
		})
	}
	return files
}

// ListSoundSlot returns the files present in the given slot on the decoder.
func (d *Client) ListSoundSlot(slot uint8) ([]RemoteFileInfo, error) {
	resp, err := d.httpGet(fmt.Sprintf(soundPackageListEndpoint, slot))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read listing response: %w", err)
	}
	return parseSoundListing(body), nil
}

// DeleteSoundFile deletes a single file from the given slot on the decoder.
func (d *Client) DeleteSoundFile(slot uint8, filename string) error {
	resp, err := d.httpGet(fmt.Sprintf(soundPackageDeleteFileEndpoint, slot, filename))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("delete %q failed with HTTP %d", filename, resp.StatusCode)
	}
	return nil
}

// UploadSoundFile uploads a file to the given slot on the decoder.
func (d *Client) UploadSoundFile(slot uint8, filename string, content io.Reader) error {
	data, err := io.ReadAll(content)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}

	url := d.baseURL + fmt.Sprintf(soundPackageUploadEndpoint, slot, filename)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to build upload request for %q: %w", filename, err)
	}
	req.Header.Set("Content-Type", "multipart/form-data")

	resp, err := d.client.Do(req)
	if err != nil {
		return fmt.Errorf("upload %q failed: %w", filename, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("upload %q failed with HTTP %d", filename, resp.StatusCode)
	}
	return nil
}
