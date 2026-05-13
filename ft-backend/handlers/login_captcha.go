package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

type loginCaptchaEntry struct {
	answer int
	exp    time.Time
}

var loginCaptchaMu sync.Mutex
var loginCaptchaStore = make(map[string]loginCaptchaEntry)

// issueLoginCaptcha creates a short-lived numeric captcha (single-instance memory store).
// For multi-instance deployments, replace with Redis-backed storage.
func issueLoginCaptcha() (id string, challenge string) {
	a, _ := rand.Int(rand.Reader, big.NewInt(9))
	b, _ := rand.Int(rand.Reader, big.NewInt(9))
	na, nb := int(a.Int64())+1, int(b.Int64())+1
	ans := na + nb
	buf := make([]byte, 16)
	_, _ = rand.Read(buf)
	id = hex.EncodeToString(buf)
	challenge = fmt.Sprintf("%d + %d = ?", na, nb)

	loginCaptchaMu.Lock()
	defer loginCaptchaMu.Unlock()
	now := time.Now()
	for k, v := range loginCaptchaStore {
		if now.After(v.exp) {
			delete(loginCaptchaStore, k)
		}
	}
	loginCaptchaStore[id] = loginCaptchaEntry{answer: ans, exp: time.Now().Add(5 * time.Minute)}
	return id, challenge
}

func verifyAndConsumeLoginCaptcha(id, answer string) bool {
	id = strings.TrimSpace(id)
	answer = strings.TrimSpace(answer)
	if id == "" || answer == "" {
		return false
	}
	loginCaptchaMu.Lock()
	ent, ok := loginCaptchaStore[id]
	delete(loginCaptchaStore, id)
	loginCaptchaMu.Unlock()
	if !ok || time.Now().After(ent.exp) {
		return false
	}
	var userAns int
	if _, err := fmt.Sscanf(answer, "%d", &userAns); err != nil {
		return false
	}
	return userAns == ent.answer
}
