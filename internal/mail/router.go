package mail

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/steveyegge/gastown/internal/tmux"
)

// Router handles message delivery and address resolution.
type Router struct {
	townRoot string
	tmux     *tmux.Tmux
}

// NewRouter creates a new mail router.
func NewRouter(townRoot string) *Router {
	return &Router{
		townRoot: townRoot,
		tmux:     tmux.NewTmux(),
	}
}

// Send delivers a message to its recipient.
func (r *Router) Send(msg *Message) error {
	// Resolve recipient mailbox path
	mailboxPath, err := r.ResolveMailbox(msg.To)
	if err != nil {
		return fmt.Errorf("resolving address '%s': %w", msg.To, err)
	}

	// Append to mailbox
	mailbox := NewMailbox(mailboxPath)
	if err := mailbox.Append(msg); err != nil {
		return fmt.Errorf("delivering message: %w", err)
	}

	// Optionally notify if recipient is a polecat with active session
	if isPolecat(msg.To) && msg.Priority == PriorityHigh {
		r.notifyPolecat(msg)
	}

	return nil
}

// ResolveMailbox converts an address to a mailbox file path.
//
// Address formats:
//   - mayor/           → <town>/mayor/mail/inbox.jsonl
//   - <rig>/refinery   → <town>/<rig>/refinery/mail/inbox.jsonl
//   - <rig>/<polecat>  → <town>/<rig>/polecats/<polecat>/mail/inbox.jsonl
//   - <rig>/           → <town>/<rig>/mail/inbox.jsonl (rig broadcast)
func (r *Router) ResolveMailbox(address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return "", fmt.Errorf("empty address")
	}

	// Mayor
	if address == "mayor/" || address == "mayor" {
		return filepath.Join(r.townRoot, "mayor", "mail", "inbox.jsonl"), nil
	}

	// Parse rig/target
	parts := strings.SplitN(address, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid address format: %s", address)
	}

	rig := parts[0]
	target := parts[1]

	// Rig broadcast (empty target or just /)
	if target == "" {
		return filepath.Join(r.townRoot, rig, "mail", "inbox.jsonl"), nil
	}

	// Refinery
	if target == "refinery" {
		return filepath.Join(r.townRoot, rig, "refinery", "mail", "inbox.jsonl"), nil
	}

	// Polecat
	return filepath.Join(r.townRoot, rig, "polecats", target, "mail", "inbox.jsonl"), nil
}

// GetMailbox returns a Mailbox for the given address.
func (r *Router) GetMailbox(address string) (*Mailbox, error) {
	path, err := r.ResolveMailbox(address)
	if err != nil {
		return nil, err
	}
	return NewMailbox(path), nil
}

// notifyPolecat sends a notification to a polecat's tmux session.
func (r *Router) notifyPolecat(msg *Message) error {
	// Parse rig/polecat from address
	parts := strings.SplitN(msg.To, "/", 2)
	if len(parts) != 2 {
		return nil
	}

	rig := parts[0]
	polecat := parts[1]

	// Generate session name (matches session.Manager)
	sessionID := fmt.Sprintf("gt-%s-%s", rig, polecat)

	// Check if session exists
	hasSession, err := r.tmux.HasSession(sessionID)
	if err != nil || !hasSession {
		return nil // No active session, skip notification
	}

	// Inject notification
	notification := fmt.Sprintf("[MAIL] %s", msg.Subject)
	return r.tmux.SendKeys(sessionID, notification)
}

// isPolecat checks if an address points to a polecat.
func isPolecat(address string) bool {
	// Not mayor, not refinery, has rig/name format
	if strings.HasPrefix(address, "mayor") {
		return false
	}

	parts := strings.SplitN(address, "/", 2)
	if len(parts) != 2 {
		return false
	}

	target := parts[1]
	return target != "" && target != "refinery"
}
