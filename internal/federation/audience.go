package federation

import (
	"context"
	"log"
	"splitter/internal/db"
	"time"
)

const publicAudience = "https://www.w3.org/ns/activitystreams#Public"

func applyAudience(activity *Activity, to []string) {
	activity.To = append([]string{}, to...)
	activity.CC = nil
	if note, ok := activity.Object.(Note); ok {
		note.To = append([]string{}, to...)
		note.CC = nil
		activity.Object = note
	}
	if note, ok := activity.Object.(map[string]interface{}); ok {
		note["to"] = to
		delete(note, "cc")
	}
}

// DeliverPostToAudience never sends restricted posts to unrelated/shared inboxes.
// Discovery of another peer is explicit; there is no synthetic wake-up traffic.
func DeliverPostToAudience(activity *Activity, authorDID, visibility string) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if db.GetDB() == nil {
		return
	}
	query := `SELECT DISTINCT ra.actor_uri,ra.inbox_url FROM follows f JOIN remote_actors ra ON (ra.actor_uri=f.follower_did OR 'did:key:'||ra.username||'@'||ra.domain=f.follower_did) WHERE f.following_did=$1 AND f.status='accepted' LIMIT 50`
	if visibility == "circle" || visibility == "private" {
		query = `SELECT DISTINCT ra.actor_uri,ra.inbox_url FROM circle_members cm JOIN users owner ON owner.id=cm.owner_id JOIN users member ON member.id=cm.member_id JOIN remote_actors ra ON (ra.actor_uri=member.did OR (ra.username=member.username AND ra.domain=member.instance_domain)) WHERE owner.did=$1 LIMIT 50`
	}
	rows, err := db.GetDB().Query(ctx, query, authorDID)
	if err != nil {
		log.Printf("[Federation] audience lookup failed")
		return
	}
	var actors, inboxes []string
	seen := map[string]bool{}
	for rows.Next() {
		var actor, inbox string
		if rows.Scan(&actor, &inbox) == nil && inbox != "" {
			actors = append(actors, actor)
			if !seen[inbox] {
				seen[inbox] = true
				inboxes = append(inboxes, inbox)
			}
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return
	}
	if visibility == "public" || visibility == "" {
		applyAudience(activity, []string{publicAudience})
		for domain, base := range InstanceURLMap {
			if domain != GetInstanceDomain() && base != "" {
				inbox := base + "/ap/shared-inbox"
				if !seen[inbox] && len(inboxes) < 50 {
					seen[inbox] = true
					inboxes = append(inboxes, inbox)
				}
			}
		}
	} else {
		if note, ok := activity.Object.(Note); ok {
			note.Visibility = visibility
			activity.Object = note
		}
		applyAudience(activity, actors)
	}
	for _, inbox := range inboxes {
		if ctx.Err() != nil {
			return
		}
		if err := DeliverActivity(activity, inbox); err != nil {
			log.Printf("[Federation] delivery queued or failed")
		}
	}
}
