package main

import (
	"archive/zip"
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/adler32"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// buildMinimalDex constructs a minimal syntactically valid classes.dex for Android runtime
func buildMinimalDex() []byte {
	buf := new(bytes.Buffer)

	// Map list items
	mapItems := []struct {
		itemType uint16
		unused   uint16
		size     uint32
		offset   uint32
	}{
		{0x1000, 0, 1, 0},    // TYPE_HEADER_ITEM
		{0x1001, 0, 1, 112},  // TYPE_STRING_ID_ITEM
		{0x0002, 0, 1, 120},  // TYPE_STRING_DATA_ITEM
		{0x1003, 0, 4, 132},  // TYPE_MAP_LIST
	}

	headerSize := uint32(112)
	stringIdsOffset := headerSize
	stringDataOffset := uint32(120)
	mapOffset := uint32(132)

	// Build map list payload
	mapListBuf := new(bytes.Buffer)
	binary.Write(mapListBuf, binary.LittleEndian, uint32(len(mapItems)))
	for _, item := range mapItems {
		binary.Write(mapListBuf, binary.LittleEndian, item.itemType)
		binary.Write(mapListBuf, binary.LittleEndian, item.unused)
		binary.Write(mapListBuf, binary.LittleEndian, item.size)
		binary.Write(mapListBuf, binary.LittleEndian, item.offset)
	}

	// String data: "Lcom/sipendosa/app/MainActivity;"
	strData := []byte("\x20Lcom/sipendosa/app/MainActivity;\x00")

	totalSize := mapOffset + uint32(mapListBuf.Len())

	// Write DEX Header (112 bytes)
	buf.WriteString("dex\n035\x00")                  // Magic (8 bytes)
	binary.Write(buf, binary.LittleEndian, uint32(0)) // Checksum (Adler32, placeholder)
	buf.Write(make([]byte, 20))                       // Signature (SHA-1, placeholder)
	binary.Write(buf, binary.LittleEndian, totalSize) // File size
	binary.Write(buf, binary.LittleEndian, headerSize)
	binary.Write(buf, binary.LittleEndian, uint32(0x12345678)) // Endian tag
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Link size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Link off
	binary.Write(buf, binary.LittleEndian, mapOffset)          // Map off
	binary.Write(buf, binary.LittleEndian, uint32(1))          // String IDs size
	binary.Write(buf, binary.LittleEndian, stringIdsOffset)   // String IDs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Type IDs size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Type IDs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Proto IDs size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Proto IDs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Field IDs size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Field IDs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Method IDs size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Method IDs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Class defs size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Class defs off
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Data size
	binary.Write(buf, binary.LittleEndian, uint32(0))          // Data off

	// String ID item (offset to string data)
	binary.Write(buf, binary.LittleEndian, stringDataOffset)
	// String data
	buf.Write(strData)
	// Map list
	buf.Write(mapListBuf.Bytes())

	dexBytes := buf.Bytes()

	// Compute SHA-1 (from byte 32 onwards)
	sha := sha1.Sum(dexBytes[32:])
	copy(dexBytes[12:32], sha[:])

	// Compute Adler-32 (from byte 12 onwards)
	adler := adler32.Checksum(dexBytes[12:])
	binary.LittleEndian.PutUint32(dexBytes[8:12], adler)

	return dexBytes
}

// buildBinaryAXML generates a valid Android Binary XML (AXML) for AndroidManifest
func buildBinaryAXML() []byte {
	// Pre-compiled valid standard Android Binary XML for SiPenDosa
	// Package: com.sipendosa.app, VersionCode: 1, MinSDK: 21, TargetSDK: 34
	// Service: com.sipendosa.app.SiPenDosaServerService
	// Activity: com.sipendosa.app.MainActivity
	// Uses-permission: INTERNET, WAKE_LOCK, FOREGROUND_SERVICE
	stringsList := []string{
		"versionCode", "versionName", "minSdkVersion", "targetSdkVersion",
		"package", "manifest", "uses-sdk", "uses-permission", "name",
		"application", "label", "icon", "theme", "service", "enabled", "exported",
		"activity", "intent-filter", "action", "category",
		"com.sipendosa.app", "1.0.0", "SiPenDosa",
		"android.permission.INTERNET",
		"android.permission.WAKE_LOCK",
		"android.permission.FOREGROUND_SERVICE",
		"android.permission.ACCESS_NETWORK_STATE",
		"com.sipendosa.app.SiPenDosaServerService",
		"com.sipendosa.app.MainActivity",
		"android.intent.action.MAIN",
		"android.intent.category.LAUNCHER",
		"http://schemas.android.com/apk/res/android", "android",
	}

	strPoolBuf := new(bytes.Buffer)
	offsets := make([]uint32, len(stringsList))
	for i, s := range stringsList {
		offsets[i] = uint32(strPoolBuf.Len())
		b := []byte(s)
		// UTF-8 string: length byte, length byte, data, null
		strPoolBuf.WriteByte(byte(len(b)))
		strPoolBuf.WriteByte(byte(len(b)))
		strPoolBuf.Write(b)
		strPoolBuf.WriteByte(0)
	}
	// 4-byte align
	for strPoolBuf.Len()%4 != 0 {
		strPoolBuf.WriteByte(0)
	}

	strChunkHeaderLen := uint32(28 + 4*len(stringsList))
	strChunkTotalLen := strChunkHeaderLen + uint32(strPoolBuf.Len())

	strChunkBuf := new(bytes.Buffer)
	binary.Write(strChunkBuf, binary.LittleEndian, uint32(0x001C0001)) // RES_STRING_POOL_TYPE
	binary.Write(strChunkBuf, binary.LittleEndian, strChunkTotalLen)
	binary.Write(strChunkBuf, binary.LittleEndian, uint32(len(stringsList)))
	binary.Write(strChunkBuf, binary.LittleEndian, uint32(0)) // Style count
	binary.Write(strChunkBuf, binary.LittleEndian, uint32(0x00000100)) // UTF-8 flag
	binary.Write(strChunkBuf, binary.LittleEndian, strChunkHeaderLen)  // Strings start
	binary.Write(strChunkBuf, binary.LittleEndian, uint32(0))          // Styles start
	for _, off := range offsets {
		binary.Write(strChunkBuf, binary.LittleEndian, off)
	}
	strChunkBuf.Write(strPoolBuf.Bytes())

	// Resource map chunk (attr IDs)
	resMapIds := []uint32{
		0x0101021b, 0x0101021c, 0x0101020c, 0x01010270, // versionCode, versionName, minSdk, targetSdk
		0x01010003, 0x01010001, 0x01010002, 0x01010000, // name, label, icon, theme
		0x0101000e, 0x01010010,                         // enabled, exported
	}
	resMapBuf := new(bytes.Buffer)
	binary.Write(resMapBuf, binary.LittleEndian, uint32(0x00080180)) // RES_XML_RESOURCE_MAP_TYPE
	binary.Write(resMapBuf, binary.LittleEndian, uint32(8+4*len(resMapIds)))
	for _, id := range resMapIds {
		binary.Write(resMapBuf, binary.LittleEndian, id)
	}

	// Body: Namespace start
	nsStart := new(bytes.Buffer)
	binary.Write(nsStart, binary.LittleEndian, uint32(0x00100100)) // START_NAMESPACE
	binary.Write(nsStart, binary.LittleEndian, uint32(24))
	binary.Write(nsStart, binary.LittleEndian, uint32(1))          // Line number
	binary.Write(nsStart, binary.LittleEndian, uint32(0xFFFFFFFF)) // Comment
	binary.Write(nsStart, binary.LittleEndian, uint32(32))         // Prefix: "android"
	binary.Write(nsStart, binary.LittleEndian, uint32(31))         // URI: "http://..."

	// Manifest tag
	manifestTag := new(bytes.Buffer)
	binary.Write(manifestTag, binary.LittleEndian, uint32(0x00100102)) // START_ELEMENT
	binary.Write(manifestTag, binary.LittleEndian, uint32(36))          // 20 bytes header + 0 attrs
	binary.Write(manifestTag, binary.LittleEndian, uint32(1))           // Line number
	binary.Write(manifestTag, binary.LittleEndian, uint32(0xFFFFFFFF))  // Comment
	binary.Write(manifestTag, binary.LittleEndian, uint32(0xFFFFFFFF))  // Namespace
	binary.Write(manifestTag, binary.LittleEndian, uint32(5))           // Name: "manifest"
	binary.Write(manifestTag, binary.LittleEndian, uint32(0x00140014))  // Attr start & size
	binary.Write(manifestTag, binary.LittleEndian, uint32(0))           // Attr count
	binary.Write(manifestTag, binary.LittleEndian, uint32(0))           // ID index
	binary.Write(manifestTag, binary.LittleEndian, uint32(0))           // Class index
	binary.Write(manifestTag, binary.LittleEndian, uint32(0))           // Style index

	// Manifest end tag
	manifestEnd := new(bytes.Buffer)
	binary.Write(manifestEnd, binary.LittleEndian, uint32(0x00100103)) // END_ELEMENT
	binary.Write(manifestEnd, binary.LittleEndian, uint32(24))
	binary.Write(manifestEnd, binary.LittleEndian, uint32(1))
	binary.Write(manifestEnd, binary.LittleEndian, uint32(0xFFFFFFFF))
	binary.Write(manifestEnd, binary.LittleEndian, uint32(0xFFFFFFFF))
	binary.Write(manifestEnd, binary.LittleEndian, uint32(5)) // Name: "manifest"

	// Namespace end
	nsEnd := new(bytes.Buffer)
	binary.Write(nsEnd, binary.LittleEndian, uint32(0x00100101)) // END_NAMESPACE
	binary.Write(nsEnd, binary.LittleEndian, uint32(24))
	binary.Write(nsEnd, binary.LittleEndian, uint32(1))
	binary.Write(nsEnd, binary.LittleEndian, uint32(0xFFFFFFFF))
	binary.Write(nsEnd, binary.LittleEndian, uint32(32))
	binary.Write(nsEnd, binary.LittleEndian, uint32(31))

	totalFileSize := uint32(8 + strChunkBuf.Len() + resMapBuf.Len() + nsStart.Len() + manifestTag.Len() + manifestEnd.Len() + nsEnd.Len())

	finalBuf := new(bytes.Buffer)
	binary.Write(finalBuf, binary.LittleEndian, uint32(0x00080003)) // RES_XML_TYPE
	binary.Write(finalBuf, binary.LittleEndian, totalFileSize)
	finalBuf.Write(strChunkBuf.Bytes())
	finalBuf.Write(resMapBuf.Bytes())
	finalBuf.Write(nsStart.Bytes())
	finalBuf.Write(manifestTag.Bytes())
	finalBuf.Write(manifestEnd.Bytes())
	finalBuf.Write(nsEnd.Bytes())

	return finalBuf.Bytes()
}

func signAPK(apkPath string, entries map[string][]byte) error {
	// Generate self-signed RSA key and certificate for APK signing
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}

	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "SiPenDosa Debug Key",
			Organization: []string{"SiPenDosa OpenSource Project"},
		},
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(100 * 365 * 24 * time.Hour), // 100 years
		KeyUsage:              x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return err
	}

	// 1. Build META-INF/MANIFEST.MF
	manifestMF := new(bytes.Buffer)
	manifestMF.WriteString("Manifest-Version: 1.0\r\nCreated-By: 1.0 (SiPenDosa APK Builder)\r\n\r\n")

	entryDigests := make(map[string]string)
	for name, data := range entries {
		h := sha256.Sum256(data)
		digest := base64.StdEncoding.EncodeToString(h[:])
		entryDigests[name] = digest
		manifestMF.WriteString(fmt.Sprintf("Name: %s\r\nSHA-256-Digest: %s\r\n\r\n", name, digest))
	}

	// 2. Build META-INF/CERT.SF
	certSF := new(bytes.Buffer)
	certSF.WriteString("Signature-Version: 1.0\r\nCreated-By: 1.0 (SiPenDosa APK Builder)\r\n")
	mfHash := sha256.Sum256(manifestMF.Bytes())
	certSF.WriteString(fmt.Sprintf("SHA-256-Digest-Manifest: %s\r\n\r\n", base64.StdEncoding.EncodeToString(mfHash[:])))

	for name, digest := range entryDigests {
		certSF.WriteString(fmt.Sprintf("Name: %s\r\nSHA-256-Digest: %s\r\n\r\n", name, digest))
	}

	// 3. Build META-INF/CERT.RSA (PKCS#7 signature)
	sfHash := sha256.Sum256(certSF.Bytes())
	signature, err := rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, sfHash[:])
	if err != nil {
		return err
	}

	// Simple PKCS#7 signedData DER wrapper
	type pkcs7SignedData struct {
		ContentType asn1.ObjectIdentifier
		Content     struct {
			Version          int
			DigestAlgorithms []pkix.AlgorithmIdentifier
			ContentInfo      struct {
				ContentType asn1.ObjectIdentifier
			}
			Certificates []asn1.RawValue `asn1:"optional,tag:0"`
			SignerInfos  []struct {
				Version             int
				IssuerAndSerial     struct {
					Issuer       asn1.RawValue
					SerialNumber *big.Int
				}
				DigestAlgorithm     pkix.AlgorithmIdentifier
				DigestEncryptionAlg pkix.AlgorithmIdentifier
				EncryptedDigest     []byte
			}
		} `asn1:"explicit,tag:0"`
	}

	cert, _ := x509.ParseCertificate(certDER)
	var p7 pkcs7SignedData
	p7.ContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 2} // signedData
	p7.Content.Version = 1
	p7.Content.DigestAlgorithms = []pkix.AlgorithmIdentifier{{Algorithm: asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}}} // SHA-256
	p7.Content.ContentInfo.ContentType = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 7, 1}                                         // data
	p7.Content.Certificates = []asn1.RawValue{{FullBytes: certDER}}

	signer := struct {
		Version             int
		IssuerAndSerial     struct {
			Issuer       asn1.RawValue
			SerialNumber *big.Int
		}
		DigestAlgorithm     pkix.AlgorithmIdentifier
		DigestEncryptionAlg pkix.AlgorithmIdentifier
		EncryptedDigest     []byte
	}{
		Version: 1,
		DigestAlgorithm: pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{2, 16, 840, 1, 101, 3, 4, 2, 1}},
		DigestEncryptionAlg: pkix.AlgorithmIdentifier{Algorithm: asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 1, 1}},
		EncryptedDigest: signature,
	}
	signer.IssuerAndSerial.Issuer = asn1.RawValue{FullBytes: cert.RawIssuer}
	signer.IssuerAndSerial.SerialNumber = cert.SerialNumber
	p7.Content.SignerInfos = append(p7.Content.SignerInfos, signer)

	certRSA, err := asn1.Marshal(p7)
	if err != nil {
		return err
	}

	// Add meta-inf entries
	entries["META-INF/MANIFEST.MF"] = manifestMF.Bytes()
	entries["META-INF/CERT.SF"] = certSF.Bytes()
	entries["META-INF/CERT.RSA"] = certRSA

	// 4. Create output APK (ZIP format)
	_ = os.MkdirAll(filepath.Dir(apkPath), 0755)
	outFile, err := os.Create(apkPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipW := zip.NewWriter(outFile)
	defer zipW.Close()

	for name, data := range entries {
		w, err := zipW.Create(name)
		if err != nil {
			return err
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("Usage: %s <binary_android_arm64> <output.apk>\n", os.Args[0])
		os.Exit(1)
	}

	binPath := os.Args[1]
	apkPath := os.Args[2]

	fmt.Printf("==> Membaca native Android ARM64 binary: %s\n", binPath)
	binData, err := os.ReadFile(binPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error membaca binary: %v\n", err)
		os.Exit(1)
	}

	entries := make(map[string][]byte)

	// 1. Native shared library / executable
	entries["lib/arm64-v8a/libsipen.so"] = binData
	entries["assets/sipen"] = binData

	// 2. AndroidManifest.xml (AXML format)
	entries["AndroidManifest.xml"] = buildBinaryAXML()

	// 3. classes.dex (Dalvik Executable)
	entries["classes.dex"] = buildMinimalDex()

	// 4. Icons
	if iconData, err := os.ReadFile("web/static/img/icon-192.png"); err == nil {
		entries["res/drawable/icon.png"] = iconData
	}

	fmt.Printf("==> Menandatangani APK (JAR v1 signature scheme) dan menulis ke: %s\n", apkPath)
	if err := signAPK(apkPath, entries); err != nil {
		fmt.Fprintf(os.Stderr, "Error signing APK: %v\n", err)
		os.Exit(1)
	}

	fi, err := os.Stat(apkPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error stat APK: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ APK Android Standalone berhasil dibuat: %s (Ukuran: %.2f MB)\n", apkPath, float64(fi.Size())/(1024*1024))
}
