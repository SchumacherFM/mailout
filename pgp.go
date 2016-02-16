package mailout

//import (
//	"archive/zip"
//	"fmt"
//	"log"
//	"os"
//	"time"
//
//	"golang.org/x/crypto/openpgp"
//	"golang.org/x/crypto/openpgp/armor"
//	"golang.org/x/crypto/openpgp/packet"
//	"golang.org/x/crypto/ssh/terminal"
//)
//
//func encrypt() {
//
//	//// open ascii armored private key
//	//from, err := os.Open("my.asc.key")
//	//logging.CheckFatal(err)
//	//defer from.Close()
//	//
//	//// decode armor and check key type
//	//fromBlock, err := armor.Decode(from)
//	//logging.CheckFatal(err)
//	//
//	//if fromBlock.Type != openpgp.PrivateKeyType {
//	//	logging.CheckFatal(fmt.Errorf("from key type:%s", fromBlock.Type))
//	//}
//	//
//	//// parse and decrypt decoded key
//	//fromReader := packet.NewReader(fromBlock.Body)
//	//fromEntity, err := openpgp.ReadEntity(fromReader)
//	//logging.CheckFatal(err)
//	//
//	//log.Println("Enter Key Passphrase:")
//	//pw, err := terminal.ReadPassword(0)
//	//logging.CheckFatal(err)
//	//
//	//err = fromEntity.PrivateKey.Decrypt(pw)
//	//logging.CheckFatal(err)
//
//	// open destination key (no ascii armor here)
//	to, err := os.Open("mkd.pubkey")
//	logging.CheckFatal(err)
//	defer to.Close()
//
//	toReader := packet.NewReader(to)
//	toEntity, err := openpgp.ReadEntity(toReader)
//	logging.CheckFatal(err)
//
//	log.Printf("to: %x", toEntity.PrimaryKey.Fingerprint)
//	//log.Printf("from: %x", fromEntity.PrimaryKey.Fingerprint)
//
//	// output file
//	out, err := os.Create("out.enc")
//	logging.CheckFatal(err)
//	defer out.Close()
//
//	hints := &openpgp.FileHints{
//		IsBinary: true,
//		FileName: "test.zip",
//		ModTime:  time.Now(),
//	}
//
//	// prepare encryption pipe
//	encOut, err := openpgp.Encrypt(out, []*openpgp.Entity{toEntity}, nil, hints, nil)
//	logging.CheckFatal(err)
//
//	// for fun, lets write a zip file to it created inline
//	zipW := zip.NewWriter(encOut)
//
//	t1, err := zipW.Create("test1.de.txt")
//	logging.CheckFatal(err)
//	fmt.Fprintln(t1, "Hallo Welt")
//
//	t2, err := zipW.Create("test1.en.txt")
//	logging.CheckFatal(err)
//	fmt.Fprintln(t2, "Hello World - the 2nd")
//
//	logging.CheckFatal(zipW.Flush())
//	logging.CheckFatal(zipW.Close())
//
//	// close the encPipe to finish the process
//	logging.CheckFatal(encOut.Close())
//}
//
//
//func encTest(secretString string) (string, error) {
//	log.Println("Secret to hide:", secretString)
//	log.Println("Public Keyring:", publicKeyring)
//
//	// Read in public key
//	keyringFileBuffer, _ := os.Open(publicKeyring)
//	defer keyringFileBuffer.Close()
//	entityList, err := openpgp.ReadKeyRing(keyringFileBuffer)
//	if err != nil {
//		return "", err
//	}
//
//	// encrypt string
//	buf := new(bytes.Buffer)
//	w, err := openpgp.Encrypt(buf, entityList, nil, nil, nil)
//	if err != nil {
//		return "", err
//	}
//	_, err = w.Write([]byte(mySecretString))
//	if err != nil {
//		return "", err
//	}
//	err = w.Close()
//	if err != nil {
//		return "", err
//	}
//
//	// Encode to base64
//	bytes, err := ioutil.ReadAll(buf)
//	if err != nil {
//		return "", err
//	}
//	encStr := base64.StdEncoding.EncodeToString(bytes)
//
//	// Output encrypted/encoded string
//	log.Println("Encrypted Secret:", encStr)
//
//	return encStr, nil
//}
