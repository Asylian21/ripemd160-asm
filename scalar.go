package ripemd160mb

import (
	"encoding/binary"
	"math/bits"
)

const (
	init0 = 0x67452301
	init1 = 0xefcdab89
	init2 = 0x98badcfe
	init3 = 0x10325476
	init4 = 0xc3d2e1f0
)

// The RIPEMD-160 compression structure and constants are from the RIPEMD-160
// specification and mirror golang.org/x/crypto/ripemd160.

type digest struct {
	s   [5]uint32
	x   [BlockSize]byte
	nx  int
	len uint64
}

func (d *digest) Reset() {
	d.s[0] = init0
	d.s[1] = init1
	d.s[2] = init2
	d.s[3] = init3
	d.s[4] = init4
	d.nx = 0
	d.len = 0
}

func (d *digest) Size() int { return Size }

func (d *digest) BlockSize() int { return BlockSize }

func (d *digest) Write(p []byte) (int, error) {
	nn := len(p)
	d.len += uint64(nn)
	if d.nx > 0 {
		n := copy(d.x[d.nx:], p)
		d.nx += n
		if d.nx == BlockSize {
			block(d, d.x[:])
			d.nx = 0
		}
		p = p[n:]
	}
	if len(p) >= BlockSize {
		n := len(p) &^ (BlockSize - 1)
		block(d, p[:n])
		p = p[n:]
	}
	if len(p) > 0 {
		d.nx = copy(d.x[:], p)
	}
	return nn, nil
}

func (d *digest) Sum(in []byte) []byte {
	d0 := *d
	var hash [Size]byte
	d0.checkSum(hash[:])
	return append(in, hash[:]...)
}

func (d *digest) checkSum(out []byte) {
	lenBits := d.len << 3
	var tmp [64]byte
	tmp[0] = 0x80
	if d.nx < 56 {
		_, _ = d.Write(tmp[:56-d.nx])
	} else {
		_, _ = d.Write(tmp[:64+56-d.nx])
	}
	binary.LittleEndian.PutUint64(tmp[:8], lenBits)
	_, _ = d.Write(tmp[:8])

	for i, s := range d.s {
		binary.LittleEndian.PutUint32(out[i*4:], s)
	}
}

func scalarHash32(dst, src []byte, n int) {
	for i := 0; i < n; i++ {
		sum32(dst[i*Size:(i+1)*Size], src[i*32:(i+1)*32])
	}
}

func sum32(dst, src []byte) {
	var x [16]uint32
	x[0] = binary.LittleEndian.Uint32(src[0:4])
	x[1] = binary.LittleEndian.Uint32(src[4:8])
	x[2] = binary.LittleEndian.Uint32(src[8:12])
	x[3] = binary.LittleEndian.Uint32(src[12:16])
	x[4] = binary.LittleEndian.Uint32(src[16:20])
	x[5] = binary.LittleEndian.Uint32(src[20:24])
	x[6] = binary.LittleEndian.Uint32(src[24:28])
	x[7] = binary.LittleEndian.Uint32(src[28:32])
	x[8] = 0x80
	x[14] = 32 * 8

	h0, h1, h2, h3, h4 := compress(init0, init1, init2, init3, init4, &x)
	binary.LittleEndian.PutUint32(dst[0:4], h0)
	binary.LittleEndian.PutUint32(dst[4:8], h1)
	binary.LittleEndian.PutUint32(dst[8:12], h2)
	binary.LittleEndian.PutUint32(dst[12:16], h3)
	binary.LittleEndian.PutUint32(dst[16:20], h4)
}

func block(dig *digest, p []byte) {
	var x [16]uint32
	for len(p) >= BlockSize {
		for i := 0; i < 16; i++ {
			x[i] = binary.LittleEndian.Uint32(p[i*4:])
		}
		dig.s[0], dig.s[1], dig.s[2], dig.s[3], dig.s[4] = compress(
			dig.s[0], dig.s[1], dig.s[2], dig.s[3], dig.s[4], &x,
		)
		p = p[BlockSize:]
	}
}

func f1(x, y, z uint32) uint32 { return x ^ y ^ z }
func f2(x, y, z uint32) uint32 { return (x & y) | (^x & z) }
func f3(x, y, z uint32) uint32 { return (x | ^y) ^ z }
func f4(x, y, z uint32) uint32 { return (x & z) | (y & ^z) }
func f5(x, y, z uint32) uint32 { return x ^ (y | ^z) }

func compress(h0, h1, h2, h3, h4 uint32, x *[16]uint32) (uint32, uint32, uint32, uint32, uint32) {
	al, bl, cl, dl, el := h0, h1, h2, h3, h4
	ar, br, cr, dr, er := h0, h1, h2, h3, h4

	for j := 0; j < 80; j++ {
		var tl uint32
		switch {
		case j < 16:
			tl = bits.RotateLeft32(al+f1(bl, cl, dl)+x[rl[j]], int(sl[j])) + el
		case j < 32:
			tl = bits.RotateLeft32(al+f2(bl, cl, dl)+x[rl[j]]+0x5a827999, int(sl[j])) + el
		case j < 48:
			tl = bits.RotateLeft32(al+f3(bl, cl, dl)+x[rl[j]]+0x6ed9eba1, int(sl[j])) + el
		case j < 64:
			tl = bits.RotateLeft32(al+f4(bl, cl, dl)+x[rl[j]]+0x8f1bbcdc, int(sl[j])) + el
		default:
			tl = bits.RotateLeft32(al+f5(bl, cl, dl)+x[rl[j]]+0xa953fd4e, int(sl[j])) + el
		}
		al, el, dl, cl, bl = el, dl, bits.RotateLeft32(cl, 10), bl, tl

		var tr uint32
		switch {
		case j < 16:
			tr = bits.RotateLeft32(ar+f5(br, cr, dr)+x[rr[j]]+0x50a28be6, int(sr[j])) + er
		case j < 32:
			tr = bits.RotateLeft32(ar+f4(br, cr, dr)+x[rr[j]]+0x5c4dd124, int(sr[j])) + er
		case j < 48:
			tr = bits.RotateLeft32(ar+f3(br, cr, dr)+x[rr[j]]+0x6d703ef3, int(sr[j])) + er
		case j < 64:
			tr = bits.RotateLeft32(ar+f2(br, cr, dr)+x[rr[j]]+0x7a6d76e9, int(sr[j])) + er
		default:
			tr = bits.RotateLeft32(ar+f1(br, cr, dr)+x[rr[j]], int(sr[j])) + er
		}
		ar, er, dr, cr, br = er, dr, bits.RotateLeft32(cr, 10), br, tr
	}

	t := h1 + cl + dr
	h1 = h2 + dl + er
	h2 = h3 + el + ar
	h3 = h4 + al + br
	h4 = h0 + bl + cr
	h0 = t
	return h0, h1, h2, h3, h4
}

var rl = [80]uint8{
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15,
	7, 4, 13, 1, 10, 6, 15, 3, 12, 0, 9, 5, 2, 14, 11, 8,
	3, 10, 14, 4, 9, 15, 8, 1, 2, 7, 0, 6, 13, 11, 5, 12,
	1, 9, 11, 10, 0, 8, 12, 4, 13, 3, 7, 15, 14, 5, 6, 2,
	4, 0, 5, 9, 7, 12, 2, 10, 14, 1, 3, 8, 11, 6, 15, 13,
}

var rr = [80]uint8{
	5, 14, 7, 0, 9, 2, 11, 4, 13, 6, 15, 8, 1, 10, 3, 12,
	6, 11, 3, 7, 0, 13, 5, 10, 14, 15, 8, 12, 4, 9, 1, 2,
	15, 5, 1, 3, 7, 14, 6, 9, 11, 8, 12, 2, 10, 0, 4, 13,
	8, 6, 4, 1, 3, 11, 15, 0, 5, 12, 2, 13, 9, 7, 10, 14,
	12, 15, 10, 4, 1, 5, 8, 7, 6, 2, 13, 14, 0, 3, 9, 11,
}

var sl = [80]uint8{
	11, 14, 15, 12, 5, 8, 7, 9, 11, 13, 14, 15, 6, 7, 9, 8,
	7, 6, 8, 13, 11, 9, 7, 15, 7, 12, 15, 9, 11, 7, 13, 12,
	11, 13, 6, 7, 14, 9, 13, 15, 14, 8, 13, 6, 5, 12, 7, 5,
	11, 12, 14, 15, 14, 15, 9, 8, 9, 14, 5, 6, 8, 6, 5, 12,
	9, 15, 5, 11, 6, 8, 13, 12, 5, 12, 13, 14, 11, 8, 5, 6,
}

var sr = [80]uint8{
	8, 9, 9, 11, 13, 15, 15, 5, 7, 7, 8, 11, 14, 14, 12, 6,
	9, 13, 15, 7, 12, 8, 9, 11, 7, 7, 12, 7, 6, 15, 13, 11,
	9, 7, 15, 11, 8, 6, 6, 14, 12, 13, 5, 14, 13, 13, 7, 5,
	15, 5, 8, 11, 14, 14, 6, 14, 6, 9, 12, 9, 12, 5, 15, 8,
	8, 5, 12, 9, 12, 5, 14, 6, 8, 13, 6, 5, 15, 13, 11, 11,
}
