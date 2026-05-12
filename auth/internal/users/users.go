package users

import (
	"crypto/subtle"

	"my-currency-service/auth/internal/config"
)

type Store struct {
	byLogin map[string]string
}

func New(cfgUsers []config.User) *Store {
	m := make(map[string]string, len(cfgUsers))
	for _, u := range cfgUsers {
		m[u.Login] = u.Password
	}
	return &Store{byLogin: m}
}

func (s *Store) Verify(login, password string) bool {
	stored, ok := s.byLogin[login]
	if !ok {
		// Дотягиваем сравнение даже при отсутствии юзера, чтобы время
		// ответа не зависело от того, нашли мы логин или нет (mitigation
		// от user enumeration по timing-сигналу).
		subtle.ConstantTimeCompare([]byte(password), []byte(password))
		return false
	}
	return subtle.ConstantTimeCompare([]byte(stored), []byte(password)) == 1
}
