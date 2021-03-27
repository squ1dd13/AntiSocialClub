package social_club

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
)

type encryptionTable struct {
	position           byte
	decryptionPosition byte
	indices            [256]byte
}

func newEncryptionTable(seed []byte) *encryptionTable {
	table := encryptionTable{position: 0, decryptionPosition: 0}

	// Set each byte to its index.
	for i := range table.indices {
		table.indices[i] = byte(i)
	}

	sum := byte(0)

	for i := range table.indices {
		// `sum` wraps around before hitting 256 because it is a byte, so
		//  it never gets past the end of the indices.
		sum += table.indices[i] + seed[i%len(seed)]

		table.swap(sum, byte(i))
	}

	return &table
}

func (table *encryptionTable) swap(a byte, b byte) {
	oldA := table.indices[a]

	table.indices[a] = table.indices[b]
	table.indices[b] = oldA
}

func (table *encryptionTable) inverseTransform(input byte) byte {
	// These two are never > 255. Any value modification can be seen as having "%= 256" (or "&= 0xff") after it.
	table.position++
	table.decryptionPosition = table.indices[table.position] + table.decryptionPosition

	table.swap(table.decryptionPosition, table.position)

	xorIndex := table.indices[table.decryptionPosition] + table.indices[table.position]
	return input ^ table.indices[xorIndex]
}

func (table *encryptionTable) transform(index byte, input byte) byte {
	table.position += table.indices[index+1]
	table.swap(index+1, table.position)

	xorIndex := table.indices[table.position] + table.indices[index+1]
	return input ^ table.indices[xorIndex]
}

// A master key which can provide multiple other keys for various uses.
type keySalt struct {
	keyBytes []byte
}

func newKeySalt(b64 string) keySalt {
	decoded, err := base64.StdEncoding.DecodeString(b64)

	if err != nil {
		panic(err)
	}

	return keySalt{keyBytes: decoded}
}

func (key keySalt) extractKey(offset int) []byte {
	// Create an encryption table to decrypt the key bytes.
	table := newEncryptionTable(key.keyBytes[1:33])

	decrypted := make([]byte, 16)

	// Decrypt the key bytes.
	for i := range decrypted {
		decrypted[i] = table.transform(byte(i), key.keyBytes[i+offset])
	}

	return decrypted
}

// The key used to encrypt and decrypt table seeds.
func (key keySalt) tableKey() []byte {
	return key.extractKey(33)
}

// The key incorporated into the SHA digest.
func (key keySalt) shaInput() []byte {
	return key.extractKey(49)
}

func createUserAgent(game string, platform string, version string) string {
	// Generate four random bytes to use as an XOR key.
	keyBytes := []byte{
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
		byte(rand.Intn(256)),
	}

	plaintext := fmt.Sprintf("e=1,t=%s,p=%s,v=%s", game, platform, version)

	// Add the key bytes to the start of the output so the receiver can decrypt the rest.
	outputBytes := append(keyBytes, []byte(plaintext)...)

	for i := 4; i < len(outputBytes); i++ {
		// % 4 so we always get one of the key values.
		outputBytes[i] ^= outputBytes[i%4]
	}

	// Base-64 encode the bytes so we have ASCII, then add "ros" to the start so the receiver
	//  knows to expect encrypted data.
	return "ros " + base64.StdEncoding.EncodeToString(outputBytes)
}

func sha1All(slices ...[]byte) []byte {
	sha := sha1.New()

	for _, slice := range slices {
		sha.Write(slice)
	}

	return sha.Sum(make([]byte, 0, 20))
}

func decrypt(key keySalt, inputBytes []byte) (string, error) {
	if len(inputBytes) <= 20 {
		return "", errors.New("too few input bytes")
	}

	// Find the table seed from the table key and the first 16 bytes of the input.
	tableSeed := make([]byte, 16)
	tableKey := key.tableKey()

	for i := range tableSeed {
		tableSeed[i] = inputBytes[i] ^ tableKey[i]
	}

	// Create an encryption table. This will be the same as the one the sender used
	//  to encrypt the data; the same encryption table is used for encryption and
	//  decryption.
	table := newEncryptionTable(tableSeed)

	// The 4 bytes beginning at index 16 are the big-endian block size of the message.
	// These bytes are encrypted.
	blockSizeBytes := make([]byte, 4)

	for i, value := range inputBytes[16:20] {
		blockSizeBytes[i] = table.inverseTransform(value)
	}

	// This is specifically the block /data/ size, because each block has an
	//  extra 20 bytes for the SHA-1 digest of its contents, and this number
	//  does not include that extra 20.
	blockDataSize := int(binary.BigEndian.Uint32(blockSizeBytes))

	// Take 20 for the size of the SHA digest.
	totalDataSize := len(inputBytes) - 20

	// The /full/ size of a block includes its SHA digest.
	fullBlockSize := blockDataSize + 20

	blockCount := totalDataSize / fullBlockSize

	extraBytesWithSHA := totalDataSize - (blockCount * fullBlockSize)

	dataByteCount := blockCount * blockDataSize

	// If there are any extra bytes, add those to the total (but subtract the
	//  20 bytes for the SHA digest first).
	if extraBytesWithSHA > 20 {
		dataByteCount += extraBytesWithSHA - 20
	}

	plaintextBytes := make([]byte, dataByteCount)

	if len(inputBytes) <= 40 {
		return string(plaintextBytes), nil
	}

	// Offset past the encrypted table seed (16 bytes) and the block size
	//  (4 bytes).
	currentBlockOffset := 20

	plaintextOffset := 0
	unprocessedByteCount := totalDataSize

	for {
		// By default, the chunk size is the number of bytes left.
		chunkSize := unprocessedByteCount - 20

		// If a block will fit into the chunk size, we only want to decrypt a
		//  single block's worth of data instead of everything that remains.
		if fullBlockSize <= unprocessedByteCount {
			chunkSize = blockDataSize
		}

		chunk := inputBytes[currentBlockOffset : currentBlockOffset+chunkSize]

		// Hash the block to make sure it's valid. The server seems to use a
		//  slight variation on the encryption algorithm we use, which includes
		//  block support and only includes two items in the hash. The lack of
		//  block support in our encryption algorithm could be due to it being
		//  permanently disabled and therefore optimised out, or it could be
		//  that the two algorithms are genuinely different.
		// The fact that there are only two items in the hashes that come from
		//  the server supports the theory of the algorithms being different.
		digest := sha1All(chunk, key.shaInput())

		expectedDigest := inputBytes[currentBlockOffset+chunkSize : currentBlockOffset+chunkSize+20]

		// Make sure the hashes are the same.
		for i := 0; i < 20; i++ {
			if digest[i] != expectedDigest[i] {
				return "", errors.New("SHA mismatch")
			}
		}

		plaintextEndOffset := plaintextOffset + chunkSize

		if plaintextEndOffset > len(plaintextBytes) {
			return "", errors.New("decrypted block too large")
		}

		if chunkSize > 0 {
			// Decrypt each byte.
			for i := 0; i < chunkSize; i++ {
				plaintextBytes[plaintextOffset+i] = table.inverseTransform(chunk[i])
			}
		}

		// Move past the chunk data and SHA.
		currentBlockOffset += chunkSize + 20

		// Move past the new plaintext we just decrypted.
		plaintextOffset += chunkSize

		unprocessedByteCount -= chunkSize + 20

		if unprocessedByteCount <= 20 {
			break
		}
	}

	return string(plaintextBytes), nil
}

func createTableSeed(key keySalt) ([]byte, []byte) {
	// Generate the random component of the table seed.
	tableSeedRandom := make([]byte, 16)

	for i := range tableSeedRandom {
		tableSeedRandom[i] = byte(rand.Intn(256))
	}

	tableKey := key.tableKey()

	tableSeed := make([]byte, 16)

	// XOR the table key with the random bytes to create a table seed.
	// The idea here is that unless the recipient knows the table key, they
	//  won't be able to create the seed: you need to XOR the random bytes with
	//  the table key in order to find the seed, and we only send the random
	//  bytes. As such, you must know the key to find the seed.
	for i := range tableSeed {
		tableSeed[i] = tableSeedRandom[i] ^ tableKey[i]
	}

	return tableSeed, tableSeedRandom
}

func encrypt(key keySalt, plaintext string) []byte {
	seed, randomBytes := createTableSeed(key)
	table := newEncryptionTable(seed)

	// The ciphertext starts off the same as the plaintext. We will apply the encryption
	//  table's transform in order to encrypt the data.
	ciphertext := []byte(plaintext)

	for i := range ciphertext {
		ciphertext[i] = table.transform(byte(i), ciphertext[i])
	}

	// Produce a SHA digest from the random bytes we used to create the seed,
	//  the encrypted data bytes, and the SHA bytes from the key. The server
	//  uses this to check that these three inputs match up with what it
	//  knows.
	shaDigest := sha1All(randomBytes, ciphertext, key.shaInput()) //sha1.Sum(append(append(randomBytes, ciphertext...), key.shaInput()...))

	return append(append(randomBytes, ciphertext...), shaDigest[:]...)
}
