package integration

import (
	"fmt"
	"math/rand"
	"time"
)

// randomSuffix returns a short unique string based on timestamp + random digits.
func randomSuffix() string {
	return fmt.Sprintf("%d%04d", time.Now().UnixNano()%1_000_000_000, rand.Intn(10000))
}

// uniqueUsername returns a unique username like "prefix_173829384".
func uniqueUsername(prefix string) string {
	return fmt.Sprintf("%s_%s", prefix, randomSuffix())
}

// uniqueEmail returns a unique email like "prefix_173829384@test.local".
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s_%s@test.local", prefix, randomSuffix())
}

// uniqueDID returns a unique DID like "did:splitter:prefix_173829384".
func uniqueDID(prefix string) string {
	return fmt.Sprintf("did:splitter:%s_%s", prefix, randomSuffix())
}

// uniqueNonce returns a unique nonce string.
func uniqueNonce(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, randomSuffix())
}

// uniqueDeviceID returns a unique device identifier.
func uniqueDeviceID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, randomSuffix())
}

// uniqueClientMsgID returns a unique client message ID.
func uniqueClientMsgID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, randomSuffix())
}
