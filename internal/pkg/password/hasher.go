package password

type Hasher interface {
	EncodedHash(password []byte) ([]byte, error)
	Verify(password, encodedHash []byte) (ok, rehash bool, err error)
}
