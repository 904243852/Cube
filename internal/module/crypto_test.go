package module

import "testing"

func TestAesEcbCipher(t *testing.T) {
	cipher, _ := (&CryptoClient{}).CreateCipher("aes-ecb", []byte("1234567890123456"), map[string]interface{}{
		"padding": "pkcs5",
	})

	a, err := cipher.Encrypt([]byte("hello, world"))
	if err != nil {
		t.Fatal(err)
	}
	h, _ := a.ToString("hex")
	if h != "5b53492af7f959b7d22054b1287b8bf7" {
		t.Fatal("unexpected encryption")
	}

	b, err := cipher.Decrypt(a)
	if err != nil {
		t.Fatal(err)
	}
	s, _ := b.ToString("")
	if s != "hello, world" {
		t.Fatal("unexpected decryption")
	}
}

func TestHash(t *testing.T) {
	hash, _ := (&CryptoClient{}).CreateHash("sha256")

	a := hash.Sum([]byte("hello, world"))
	h, _ := a.ToString("hex")
	if h != "09ca7e4eaa6e8ae9c7d261167129184883644d07dfba7cbfbc4c8a2e08360d5b" {
		t.Fatal("unexpected sum")
	}
}

func BenchmarkHash(b *testing.B) {
	hash, _ := (&CryptoClient{}).CreateHash("sha256")

	for n := 0; n < b.N; n++ { // b.N 从 1 开始，如果用例能够在 1 秒内完成，b.N 的值则会增加（以 1 2 3 5 10 20 30 50 100... 的序列递增）并再次执行
		hash.Sum([]byte("hello, world"))
	}
}
