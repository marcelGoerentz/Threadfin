package up2date

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

const pubkey = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA2ZkUyr5n1c1iSO8fMtFl
ZV1zncySavnEOOADT+1jHAJVASipxYbm+gZ9welcAl5iYZSXd0FV7uGBbU1HogIL
pl3zfRJ4xY+TmVDRGdKfnc2KMn/7aY3JdSUfSCZ+smZVvHNRWlnzVsRoYNHMe6gm
dHg50I1IgFuhdtYY4p3e4FS4SdZVNV3BJuJTaAG8viUpsoy30DnaWIBsDpX+CPjp
7ntWZPGxDxCRl011ri+DWd3dgcClDmt6YXJAzkiSnfuNqXwENu2nhFV7wCqwMmFU
nfkRuBOYqTJ3GAsAJK+HNthxWV09p9O1n1pgLpaBNcp8UosBWXH+JEJU7UKSp2u3
bwIDAQAB
-----END PUBLIC KEY-----`

func checkSignature(filePath string) (bool, error) {
	file, err := os.Open("threadfin")
	if err != nil {
		return false, err
	}
	// Gehe zum Ende der Datei und extrahiere die letzten 256 Bytes (Signatur)#
	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Fehler beim Abrufen der Dateiinformationen:", err)
		return false, err
	}
	signatureSize := 256 // Größe der RSA Signatur in Bytes
	fileSize := fileInfo.Size()
	if fileSize < int64(signatureSize) {
		fmt.Println("Dateigröße zu klein für Signatur")
		return false, err
	}
	// Extrahiere die Signatur
	_, err = file.Seek(-int64(signatureSize), io.SeekEnd)
	if err != nil {
		fmt.Println("Fehler beim Suchen der Signatur:", err)
		return false, err
	}
	signature := make([]byte, signatureSize)
	_, err = file.Read(signature)
	if err != nil {
		fmt.Println("Fehler beim Lesen der Signatur:", err)
		return false, err
	}
	// Gehe zum Anfang der Datei und lese den Inhalt ohne die Signatur
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		fmt.Println("Fehler beim Suchen des Datei-Anfangs:", err)
		return false, err
	}
	content := make([]byte, fileSize-int64(signatureSize))
	_, err = file.Read(content)
	if err != nil {
		fmt.Println("Fehler beim Lesen des Datei-Inhalts:", err)
		return false, err
	}
	// Lade den öffentlichen Schlüssel (aus einer PEM-Datei)
	block, _ := pem.Decode([]byte(pubkey))
	if block == nil || block.Type != "PUBLIC KEY" {
		fmt.Println("Fehler beim Dekodieren des öffentlichen Schlüssels")
		return false, err
	}
	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		fmt.Println("Fehler beim Parsen des öffentlichen Schlüssels:", err)
		return false, err
	}
	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		fmt.Println("Öffentlicher Schlüssel ist kein RSA-Schlüssel")
		return false, err
	}
	// Berechne den SHA-256 Hash des Inhalts ohne die Signatur
	hash := sha256.Sum256(content)
	// Überprüfe die Signatur
	err = rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hash[:], signature)
	if err != nil {
		fmt.Println("Signaturüberprüfung fehlgeschlagen:", err)
		return false, err
	} else {
		fmt.Println("Signaturüberprüfung erfolgreich!")
		return true, nil
	}
}