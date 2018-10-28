package util

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"io"
	prand "math/rand"
	"net"
	"strconv"
)

func AESEncrypt(iv []byte, key []byte, plaintext []byte) []byte {
	block, _ := aes.NewCipher(key)
	ciphertext := make([]byte, len(plaintext))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext, plaintext)
	return ciphertext
}

func AESDecrypt(iv []byte, key []byte, ciphertext []byte) []byte {
	ret := make([]byte, len(ciphertext))
	copy(ret, ciphertext) // Because XORKeyStream is inplace, but we don't want the input to be changed
	block, _ := aes.NewCipher(key)
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ret, ret)
	return ret
}

// BtoInt converts a byte slice into int in Big Endian order
// Uint methods from binary package can be used, but they are messy
func BtoInt(b []byte) int {
	var mult uint = 1
	var sum uint
	length := uint(len(b))
	var i uint
	for i = 0; i < length; i++ {
		sum += uint(b[i]) * (mult << ((length - i - 1) * 8))
	}
	return int(sum)
}

// PsudoRandBytes returns a byte slice filled with psudorandom bytes generated by the seed
func PsudoRandBytes(length int, seed int64) []byte {
	prand.Seed(seed)
	ret := make([]byte, length)
	prand.Read(ret)
	return ret
}

// ReadTillDrain reads TLS data according to its record layer
func ReadTillDrain(conn net.Conn, buffer []byte) (n int, err error) {
	// TCP is a stream. Multiple TLS messages can arrive at the same time,
	// a single message can also be segmented due to MTU of the IP layer.
	// This function guareentees a single TLS message to be read and everything
	// else is left in the buffer.
	i, err := io.ReadFull(conn, buffer[:5])
	if err != nil {
		return
	}

	dataLength := BtoInt(buffer[3:5])
	if dataLength > len(buffer) {
		err = errors.New("Reading TLS message: message size greater than buffer. message size: " + strconv.Itoa(dataLength))
		return
	}
	left := dataLength
	readPtr := 5

	for left != 0 {
		// If left > buffer size (i.e. our message got segmented), the entire MTU is read
		// if left = buffer size, the entire buffer is all there left to read
		// if left < buffer size (i.e. multiple messages came together),
		// only the message we want is read
		i, err = io.ReadFull(conn, buffer[readPtr:readPtr+left])
		if err != nil {
			return
		}
		left -= i
		readPtr += i
	}

	n = 5 + dataLength
	return
}

// AddRecordLayer adds record layer to data
func AddRecordLayer(input []byte, typ []byte, ver []byte) []byte {
	length := make([]byte, 2)
	binary.BigEndian.PutUint16(length, uint16(len(input)))
	ret := make([]byte, 5+len(input))
	copy(ret[0:1], typ)
	copy(ret[1:3], ver)
	copy(ret[3:5], length)
	copy(ret[5:], input)
	return ret
}

// PeelRecordLayer peels off the record layer
func PeelRecordLayer(data []byte) []byte {
	ret := data[5:]
	return ret
}