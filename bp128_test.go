package bp128

import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"reflect"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
)

type intSlice reflect.Value

func (s intSlice) Len() int {
	return reflect.Value(s).Len()
}

func (s intSlice) Less(i, j int) bool {
	v := reflect.Value(s)
	return v.Index(i).Uint() < v.Index(j).Uint()
}

func (s intSlice) Swap(i, j int) {
	v := reflect.Value(s)
	ii := v.Index(i).Uint()
	jj := v.Index(j).Uint()
	v.Index(i).SetUint(jj)
	v.Index(j).SetUint(ii)
}

func testPackUnpackAsm(t *testing.T, intSize, nbits int, isDiffCode bool) bool {
	const blockSize = 128

	fpack := fpack32
	funpack := funpack32
	var sliceType interface{}
	if intSize == 64 {
		fpack = fpack64
		funpack = funpack64
		sliceType = []uint64{}
		if isDiffCode {
			fpack = fdpack64
			funpack = fdunpack64
		}
	} else if intSize == 32 {
		sliceType = []uint32{}
		if isDiffCode {
			fpack = fdpack32
			funpack = fdunpack32
		}
	}

	max := 1 << uint(nbits)
	in := makeAlignedSlice(sliceType, blockSize)
	out := makeAlignedSlice(sliceType, blockSize)
	zip := makeAlignedBytes((nbits * blockSize) / 8)

	for i := 0; i < blockSize; i++ {
		if nbits < 63 {
			in.Index(i).SetUint(uint64(rand.Intn(max)))
		} else {
			in.Index(i).SetUint(uint64(rand.Int63()))
		}

	}
	sort.Sort(intSlice(in))

	nslice := blockSize / intSize
	seed := makeAlignedBytes(16)
	copy(seed, convertToBytes(intSize, in.Slice(0, nslice)))

	inAddr := in.Pointer()
	outAddr := out.Pointer()

	fpack[nbits](inAddr, &zip[0], 0, &seed[0])

	copy(seed, convertToBytes(intSize, in.Slice(0, nslice)))
	funpack[nbits](&zip[0], outAddr, 0, &seed[0])

	equal := false
	for i := 0; i < blockSize; i++ {
		equal = assert.Equal(t, in.Index(i).Uint(), out.Index(i).Uint())
		if !equal {
			break
		}
	}

	return equal
}

func TestPackUnpackAsm32(t *testing.T) {
	for i := 1; i <= 32; i++ {
		if !testPackUnpackAsm(t, 32, i, false) {
			fmt.Printf("Pack-unpack for bit size %d failed\n", i)
			break
		}
	}
}

func TestPackUnpackAsm64(t *testing.T) {
	for i := 1; i <= 64; i++ {
		if !testPackUnpackAsm(t, 64, i, false) {
			fmt.Printf("Pack-unpack for bit size %d failed\n", i)
			break
		}
	}
}

func TestDeltaPackUnpackAsm32(t *testing.T) {
	for i := 1; i <= 32; i++ {
		if !testPackUnpackAsm(t, 32, i, true) {
			fmt.Printf("Pack-unpack for bit size %d failed\n", i)
			break
		}
	}
}

func TestDeltaPackUnpackAsm64(t *testing.T) {
	for i := 1; i <= 64; i++ {
		if !testPackUnpackAsm(t, 64, i, true) {
			fmt.Printf("Pack-unpack for bit size %d failed\n", i)
			break
		}
	}
}

var getData32 = func() func() []uint32 {
	f, err := os.Open("data/clustered100K.bin")
	if err != nil {
		panic(err)
	}

	buf := make([]byte, 4)
	_, err = f.Read(buf)
	if err != nil {
		panic(err)
	}

	ndata := binary.LittleEndian.Uint32(buf)
	data := make([]uint32, ndata)

	for i := range data {
		_, err = f.Read(buf)
		if err != nil {
			panic(err)
		}

		data[i] = binary.LittleEndian.Uint32(buf)
	}

	return func() []uint32 { return data }
}()

var getData64 = func() func() []uint64 {
	data := getData32()
	data64 := make([]uint64, len(data))
	for i, d := range data {
		data64[i] = uint64(d)
	}

	return func() []uint64 { return data64 }
}()

func TestPackUnpack32(t *testing.T) {
	data := getData32()

	var out []uint32
	packed := PackInts(data)
	UnpackInts(packed, &out)

	assert.Equal(t, data, out)
}

func TestPackUnpack64(t *testing.T) {
	data := getData64()

	var out []uint64
	packed := PackInts(data)
	UnpackInts(packed, &out)

	assert.Equal(t, data, out)
}

func TestDeltaPackUnpack32(t *testing.T) {
	data := getData32()

	var out []uint32
	packed := DeltaPackInts(data)
	UnpackInts(packed, &out)

	assert.Equal(t, data, out)
}

func TestDeltaPackUnpack64(t *testing.T) {
	data := getData64()

	var out []uint64
	packed := DeltaPackInts(data)
	UnpackInts(packed, &out)

	assert.Equal(t, data, out)
}

func TestPackedIntsEncDec(t *testing.T) {
	data := getData32()

	packed1 := PackInts(data)
	enc, _ := packed1.GobEncode()

	packed2 := &PackedInts{}
	packed2.GobDecode(enc)

	var out []uint32
	UnpackInts(packed2, &out)
	assert.Equal(t, data, out)
}

func TestDeltaPackedIntsEncDec(t *testing.T) {
	data := getData32()

	packed1 := DeltaPackInts(data)
	enc, _ := packed1.GobEncode()

	packed2 := &PackedInts{}
	packed2.GobDecode(enc)

	var out []uint32
	UnpackInts(packed2, &out)
	assert.Equal(t, data, out)
}
