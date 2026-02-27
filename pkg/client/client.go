package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/naveenspark/grimora/pkg/domain"
)

// CreateSpellRequest is the payload for creating a new spell.
type CreateSpellRequest struct {
	Text    string   `json:"text"`
	Tag     string   `json:"tag"`
	Model   string   `json:"model,omitempty"`
	Stack   []string `json:"stack,omitempty"`
	Context string   `json:"context,omitempty"`
}

// Client is the Grimora API client.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates a new API client.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetMe returns the authenticated magician's profile.
func (c *Client) GetMe(ctx context.Context) (*domain.Magician, error) {
	var m domain.Magician
	if err := c.get(ctx, "/api/me", &m); err != nil {
		return nil, fmt.Errorf("client.GetMe: %w", err)
	}
	return &m, nil
}

// GetForgeStats returns the authenticated magician's forge stats.
func (c *Client) GetForgeStats(ctx context.Context) (*domain.ForgeStats, error) {
	var stats domain.ForgeStats
	if err := c.get(ctx, "/api/me/forge-stats", &stats); err != nil {
		return nil, fmt.Errorf("client.GetForgeStats: %w", err)
	}
	return &stats, nil
}

// ListSpells fetches spells with optional tag filter and sort.
func (c *Client) ListSpells(ctx context.Context, tag, sort string, limit, offset int) ([]domain.Spell, error) {
	params := url.Values{}
	if tag != "" {
		params.Set("tag", tag)
	}
	if sort != "" {
		params.Set("sort", sort)
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var spells []domain.Spell
	if err := c.get(ctx, "/api/spells?"+params.Encode(), &spells); err != nil {
		return nil, fmt.Errorf("client.ListSpells: %w", err)
	}
	return spells, nil
}

// TagStats returns spell counts and upvote totals per tag.
func (c *Client) TagStats(ctx context.Context) ([]domain.TagStat, error) {
	var stats []domain.TagStat
	if err := c.get(ctx, "/api/spells/tags", &stats); err != nil {
		return nil, fmt.Errorf("client.TagStats: %w", err)
	}
	return stats, nil
}

// SearchSpells searches spells by text query.
func (c *Client) SearchSpells(ctx context.Context, query string) ([]domain.Spell, error) {
	params := url.Values{}
	params.Set("q", query)

	var spells []domain.Spell
	if err := c.get(ctx, "/api/spells?"+params.Encode(), &spells); err != nil {
		return nil, fmt.Errorf("client.SearchSpells: %w", err)
	}
	return spells, nil
}

// GetSpell fetches a single spell by ID.
func (c *Client) GetSpell(ctx context.Context, id string) (*domain.Spell, error) {
	var spell domain.Spell
	if err := c.get(ctx, "/api/spells/"+url.PathEscape(id), &spell); err != nil {
		return nil, fmt.Errorf("client.GetSpell: %w", err)
	}
	return &spell, nil
}

// CreateSpell creates a new spell.
func (c *Client) CreateSpell(ctx context.Context, spell CreateSpellRequest) (*domain.Spell, error) {
	var created domain.Spell
	if err := c.post(ctx, "/api/spells", spell, &created); err != nil {
		return nil, fmt.Errorf("client.CreateSpell: %w", err)
	}
	return &created, nil
}

// UpvoteSpell upvotes a spell by ID.
func (c *Client) UpvoteSpell(ctx context.Context, id string) error {
	if err := c.doRequest(ctx, http.MethodPost, "/api/spells/"+url.PathEscape(id)+"/upvote", nil, nil); err != nil {
		return fmt.Errorf("client.UpvoteSpell: %w", err)
	}
	return nil
}

// RemoveUpvote removes an upvote from a spell.
func (c *Client) RemoveUpvote(ctx context.Context, id string) error {
	if err := c.doRequest(ctx, http.MethodDelete, "/api/spells/"+url.PathEscape(id)+"/upvote", nil, nil); err != nil {
		return fmt.Errorf("client.RemoveUpvote: %w", err)
	}
	return nil
}

// --- Weapon methods ---

// CreateWeaponRequest is the payload for creating a new weapon.
type CreateWeaponRequest struct {
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	RepositoryURL  string `json:"repository_url"`
	GitHubStars    int    `json:"github_stars,omitempty"`
	GitHubForks    int    `json:"github_forks,omitempty"`
	GitHubLanguage string `json:"github_language,omitempty"`
	License        string `json:"license,omitempty"`
}

// ListWeapons fetches weapons.
func (c *Client) ListWeapons(ctx context.Context, limit, offset int) ([]domain.Weapon, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var weapons []domain.Weapon
	if err := c.get(ctx, "/api/weapons?"+params.Encode(), &weapons); err != nil {
		return nil, fmt.Errorf("client.ListWeapons: %w", err)
	}
	return weapons, nil
}

// SearchWeapons searches weapons by text query.
func (c *Client) SearchWeapons(ctx context.Context, query string) ([]domain.Weapon, error) {
	params := url.Values{}
	params.Set("q", query)

	var weapons []domain.Weapon
	if err := c.get(ctx, "/api/weapons?"+params.Encode(), &weapons); err != nil {
		return nil, fmt.Errorf("client.SearchWeapons: %w", err)
	}
	return weapons, nil
}

// GetWeapon fetches a single weapon by ID.
func (c *Client) GetWeapon(ctx context.Context, id string) (*domain.Weapon, error) {
	var weapon domain.Weapon
	if err := c.get(ctx, "/api/weapons/"+url.PathEscape(id), &weapon); err != nil {
		return nil, fmt.Errorf("client.GetWeapon: %w", err)
	}
	return &weapon, nil
}

// CreateWeapon creates a new weapon.
func (c *Client) CreateWeapon(ctx context.Context, w CreateWeaponRequest) (*domain.Weapon, error) {
	var created domain.Weapon
	if err := c.post(ctx, "/api/weapons", w, &created); err != nil {
		return nil, fmt.Errorf("client.CreateWeapon: %w", err)
	}
	return &created, nil
}

// SaveWeapon saves a weapon by ID.
func (c *Client) SaveWeapon(ctx context.Context, id string) error {
	if err := c.doRequest(ctx, http.MethodPost, "/api/weapons/"+url.PathEscape(id)+"/save", nil, nil); err != nil {
		return fmt.Errorf("client.SaveWeapon: %w", err)
	}
	return nil
}

// RemoveWeaponSave removes a weapon save.
func (c *Client) RemoveWeaponSave(ctx context.Context, id string) error {
	if err := c.doRequest(ctx, http.MethodDelete, "/api/weapons/"+url.PathEscape(id)+"/save", nil, nil); err != nil {
		return fmt.Errorf("client.RemoveWeaponSave: %w", err)
	}
	return nil
}

// --- Social methods ---

// GetStream returns the activity feed.
// If followingOnly is true, only events from followed magicians are returned.
func (c *Client) GetStream(ctx context.Context, followingOnly bool, limit, offset int) ([]domain.StreamEvent, error) {
	params := url.Values{}
	if followingOnly {
		params.Set("following", "true")
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var events []domain.StreamEvent
	if err := c.get(ctx, "/api/stream?"+params.Encode(), &events); err != nil {
		return nil, fmt.Errorf("client.GetStream: %w", err)
	}
	return events, nil
}

// ListMagicians returns a paginated list of magicians with follow state.
func (c *Client) ListMagicians(ctx context.Context, limit, offset int) ([]domain.MagicianCard, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var cards []domain.MagicianCard
	if err := c.get(ctx, "/api/magicians?"+params.Encode(), &cards); err != nil {
		return nil, fmt.Errorf("client.ListMagicians: %w", err)
	}
	return cards, nil
}

// Follow follows a magician by login.
func (c *Client) Follow(ctx context.Context, login string) error {
	if err := c.doRequest(ctx, http.MethodPost, "/api/magicians/"+url.PathEscape(login)+"/follow", nil, nil); err != nil {
		return fmt.Errorf("client.Follow: %w", err)
	}
	return nil
}

// Unfollow unfollows a magician by login.
func (c *Client) Unfollow(ctx context.Context, login string) error {
	if err := c.doRequest(ctx, http.MethodDelete, "/api/magicians/"+url.PathEscape(login)+"/follow", nil, nil); err != nil {
		return fmt.Errorf("client.Unfollow: %w", err)
	}
	return nil
}

// --- Thread methods ---

// ListThreads returns the caller's DM threads.
func (c *Client) ListThreads(ctx context.Context) ([]domain.Thread, error) {
	var threads []domain.Thread
	if err := c.get(ctx, "/api/threads", &threads); err != nil {
		return nil, fmt.Errorf("client.ListThreads: %w", err)
	}
	return threads, nil
}

// StartThread creates or retrieves a DM thread with the given magician login.
func (c *Client) StartThread(ctx context.Context, login string) (*domain.Thread, error) {
	var thread domain.Thread
	if err := c.post(ctx, "/api/threads", map[string]string{"login": login}, &thread); err != nil {
		return nil, fmt.Errorf("client.StartThread: %w", err)
	}
	return &thread, nil
}

// GetMessages returns messages in a thread.
func (c *Client) GetMessages(ctx context.Context, threadID string, limit, offset int) ([]domain.Message, error) {
	params := url.Values{}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var msgs []domain.Message
	if err := c.get(ctx, "/api/threads/"+url.PathEscape(threadID)+"/messages?"+params.Encode(), &msgs); err != nil {
		return nil, fmt.Errorf("client.GetMessages: %w", err)
	}
	return msgs, nil
}

// SendMessage sends a message to a thread.
func (c *Client) SendMessage(ctx context.Context, threadID, body string) (*domain.Message, error) {
	var msg domain.Message
	if err := c.post(ctx, "/api/threads/"+url.PathEscape(threadID)+"/messages", map[string]string{"body": body}, &msg); err != nil {
		return nil, fmt.Errorf("client.SendMessage: %w", err)
	}
	return &msg, nil
}

// --- Invites ---

// ListInvites returns the authenticated magician's invite codes.
func (c *Client) ListInvites(ctx context.Context) ([]domain.Invite, error) {
	var invites []domain.Invite
	if err := c.get(ctx, "/api/invites", &invites); err != nil {
		return nil, fmt.Errorf("client.ListInvites: %w", err)
	}
	return invites, nil
}

// --- Workshop ---

// ListWorkshopProjects returns the current magician's workshop projects.
func (c *Client) ListWorkshopProjects(ctx context.Context) ([]domain.WorkshopProject, error) {
	var projects []domain.WorkshopProject
	if err := c.get(ctx, "/api/workshop", &projects); err != nil {
		return nil, fmt.Errorf("client.ListWorkshopProjects: %w", err)
	}
	return projects, nil
}

// CreateWorkshopProject creates a new workshop project.
func (c *Client) CreateWorkshopProject(ctx context.Context, name, insight string) (*domain.WorkshopProject, error) {
	var project domain.WorkshopProject
	if err := c.post(ctx, "/api/workshop", map[string]string{"name": name, "insight": insight}, &project); err != nil {
		return nil, fmt.Errorf("client.CreateWorkshopProject: %w", err)
	}
	return &project, nil
}

// UpdateWorkshopProject updates a workshop project.
func (c *Client) UpdateWorkshopProject(ctx context.Context, id, name, insight string) error {
	if err := c.doRequest(ctx, http.MethodPut, "/api/workshop/"+url.PathEscape(id), map[string]string{"name": name, "insight": insight}, nil); err != nil {
		return fmt.Errorf("client.UpdateWorkshopProject: %w", err)
	}
	return nil
}

// DeleteWorkshopProject deletes a workshop project.
func (c *Client) DeleteWorkshopProject(ctx context.Context, id string) error {
	if err := c.doRequest(ctx, http.MethodDelete, "/api/workshop/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("client.DeleteWorkshopProject: %w", err)
	}
	return nil
}

// GetLeaderboard returns ranked magicians with optional guild/city filters.
func (c *Client) GetLeaderboard(ctx context.Context, guild, city string, limit, offset int) ([]domain.LeaderboardEntry, error) {
	params := url.Values{}
	if guild != "" {
		params.Set("guild", guild)
	}
	if city != "" {
		params.Set("city", city)
	}
	params.Set("limit", strconv.Itoa(limit))
	params.Set("offset", strconv.Itoa(offset))

	var entries []domain.LeaderboardEntry
	if err := c.get(ctx, "/api/leaderboard?"+params.Encode(), &entries); err != nil {
		return nil, fmt.Errorf("client.GetLeaderboard: %w", err)
	}
	return entries, nil
}

// ListProjectUpdates returns timeline entries for a workshop project.
func (c *Client) ListProjectUpdates(ctx context.Context, projectID string) ([]domain.ProjectUpdate, error) {
	var updates []domain.ProjectUpdate
	if err := c.get(ctx, "/api/workshop/"+url.PathEscape(projectID)+"/updates", &updates); err != nil {
		return nil, fmt.Errorf("client.ListProjectUpdates: %w", err)
	}
	return updates, nil
}

// GetMagicianWorkshop returns a magician's workshop projects (public view).
func (c *Client) GetMagicianWorkshop(ctx context.Context, login string) ([]domain.WorkshopProject, error) {
	var projects []domain.WorkshopProject
	if err := c.get(ctx, "/api/magicians/"+url.PathEscape(login)+"/workshop", &projects); err != nil {
		return nil, fmt.Errorf("client.GetMagicianWorkshop: %w", err)
	}
	return projects, nil
}

// GetMagician fetches a single magician card by login.
func (c *Client) GetMagician(ctx context.Context, login string) (*domain.MagicianCard, error) {
	var card domain.MagicianCard
	if err := c.get(ctx, "/api/magicians/"+url.PathEscape(login), &card); err != nil {
		return nil, fmt.Errorf("client.GetMagician: %w", err)
	}
	return &card, nil
}

// --- Rooms ---

// ListRooms returns all non-archived chat rooms.
func (c *Client) ListRooms(ctx context.Context) ([]domain.Room, error) {
	var rooms []domain.Room
	if err := c.get(ctx, "/api/rooms", &rooms); err != nil {
		return nil, fmt.Errorf("client.ListRooms: %w", err)
	}
	return rooms, nil
}

// RoomPresence is the response from the room presence endpoint.
type RoomPresence struct {
	RoomSlug  string   `json:"room_slug"`
	Count     int      `json:"count"`
	Magicians []string `json:"magicians"`
}

// GetRoomPresence returns the live presence for a room.
func (c *Client) GetRoomPresence(ctx context.Context, slug string) (*RoomPresence, error) {
	var p RoomPresence
	if err := c.get(ctx, "/api/rooms/"+url.PathEscape(slug)+"/presence", &p); err != nil {
		return nil, fmt.Errorf("client.GetRoomPresence: %w", err)
	}
	return &p, nil
}

// GetRoomMessages returns paginated messages from a room.
func (c *Client) GetRoomMessages(ctx context.Context, slug string, before time.Time, limit int) ([]domain.RoomMessage, error) {
	params := url.Values{}
	if !before.IsZero() {
		params.Set("before", before.Format(time.RFC3339Nano))
	}
	params.Set("limit", strconv.Itoa(limit))

	var msgs []domain.RoomMessage
	if err := c.get(ctx, "/api/rooms/"+url.PathEscape(slug)+"/messages?"+params.Encode(), &msgs); err != nil {
		return nil, fmt.Errorf("client.GetRoomMessages: %w", err)
	}
	return msgs, nil
}

// ReactionCount is an emoji + count pair from the reaction counts endpoint.
type ReactionCount struct {
	Emoji string `json:"emoji"`
	Count int    `json:"count"`
}

// GetReactionCounts fetches batch reaction counts for a set of message IDs.
func (c *Client) GetReactionCounts(ctx context.Context, slug string, msgIDs []string) (map[string][]ReactionCount, error) {
	if len(msgIDs) == 0 {
		return nil, nil
	}
	ids := strings.Join(msgIDs, ",")
	var result map[string][]ReactionCount
	if err := c.get(ctx, "/api/rooms/"+url.PathEscape(slug)+"/messages/reactions?ids="+url.QueryEscape(ids), &result); err != nil {
		return nil, fmt.Errorf("client.GetReactionCounts: %w", err)
	}
	return result, nil
}

// SendRoomMessage posts a message to a chat room.
func (c *Client) SendRoomMessage(ctx context.Context, slug, body string) (*domain.RoomMessage, error) {
	var msg domain.RoomMessage
	if err := c.post(ctx, "/api/rooms/"+url.PathEscape(slug)+"/messages", map[string]string{"body": body}, &msg); err != nil {
		return nil, fmt.Errorf("client.SendRoomMessage: %w", err)
	}
	return &msg, nil
}

// JoinRoom joins a chat room.
func (c *Client) JoinRoom(ctx context.Context, slug string) error {
	if err := c.doRequest(ctx, http.MethodPost, "/api/rooms/"+url.PathEscape(slug)+"/join", nil, nil); err != nil {
		return fmt.Errorf("client.JoinRoom: %w", err)
	}
	return nil
}

// --- Telemetry ---

// TelemetryResponse is the shape of the telemetry endpoint response.
type TelemetryResponse struct {
	Total  int              `json:"total"`
	Cities []CityCountEntry `json:"cities"`
}

// CityCountEntry is a city + count pair in the telemetry response.
type CityCountEntry struct {
	City  string `json:"city"`
	Count int    `json:"count"`
}

// GetTelemetry returns platform-level telemetry (magician count + top cities).
func (c *Client) GetTelemetry(ctx context.Context) (*TelemetryResponse, error) {
	var t TelemetryResponse
	if err := c.get(ctx, "/api/telemetry", &t); err != nil {
		return nil, fmt.Errorf("client.GetTelemetry: %w", err)
	}
	return &t, nil
}

func (c *Client) post(ctx context.Context, path string, body any, out any) error {
	return c.doRequest(ctx, http.MethodPost, path, body, out)
}

func (c *Client) doRequest(ctx context.Context, method, path string, body any, out any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort close

	if resp.StatusCode >= 400 {
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max error body
		if readErr != nil {
			return &HTTPError{StatusCode: resp.StatusCode, Message: fmt.Sprintf("failed to read body: %v", readErr)}
		}
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
			return &HTTPError{StatusCode: resp.StatusCode, Message: apiErr.Error}
		}
		return &HTTPError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	if out != nil {
		if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	return c.doRequest(ctx, http.MethodGet, path, nil, out)
}
