package domain

type HashedPassword []byte

func NewHashedPassword(key []byte) HashedPassword {
	if key == nil {
		key = make([]byte, 0)
	}

	return HashedPassword(key)
}
