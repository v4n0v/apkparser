// Package apkparser parses AndroidManifest.xml and resources.arsc from Android APKs.
package apkparser

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"github.com/appflight/androidbinary"
	"image"
	"image/png"
	"io"
	"os"
	"runtime/debug"
	"strings"
)

type ApkInfo struct {
	Package     string
	VersionName string
	VersionCode int32
	Label       string
	Icon        image.Image
}
type Manifest struct {
	Package     androidbinary.String `xml:"package,attr"`
	VersionCode androidbinary.Int32  `xml:"http://schemas.android.com/apk/res/android versionCode,attr"`
	VersionName androidbinary.String `xml:"http://schemas.android.com/apk/res/android versionName,attr"`
	App         Application          `xml:"application"`
}
type Application struct {
	Icon  androidbinary.String `xml:"http://schemas.android.com/apk/res/android icon,attr"`
	Label androidbinary.String `xml:"http://schemas.android.com/apk/res/android label,attr"`
	Logo  androidbinary.String `xml:"http://schemas.android.com/apk/res/android logo,attr"`
	Name  androidbinary.String `xml:"http://schemas.android.com/apk/res/android name,attr"`
}

type ApkParser struct {
	apkPath string
	zip     *ZipReader

	encoder   ManifestEncoder
	resources *ResourceTable
}

// Calls ParseApkReader
func ParseApk(path string) (ApkInfo, error) {
	f, zipErr := os.Open(path)
	if zipErr != nil {
		return ApkInfo{}, zipErr
	}
	defer f.Close()
	return ParseApkReader(f)
}

// Parse APK's Manifest, including resolving refences to resource values.
// encoder expects an XML encoder instance, like Encoder from encoding/xml package.
//
// zipErr != nil means the APK couldn't be opened. The manifest will be parsed
// even when resourcesErr != nil, just without reference resolving.
func ParseApkReader(r io.ReadSeeker) (ApkInfo, error) {
	zip, zipErr := OpenZipReader(r)
	if zipErr != nil {
		return ApkInfo{}, zipErr
	}
	defer zip.Close()

	return ParseApkWithZip(zip)
}

// Parse APK's Manifest, including resolving refences to resource values.
// encoder expects an XML encoder instance, like Encoder from encoding/xml package.
//
// Use this if you already opened the zip with OpenZip or OpenZipReader before.
// This method will not Close() the zip.
//
// The manifest will be parsed even when resourcesErr != nil, just without reference resolving.
func ParseApkWithZip(zip *ZipReader) (ApkInfo, error) {
	var apkInfo ApkInfo
	buf := new(bytes.Buffer)
	enc := xml.NewEncoder(buf)
	enc.Indent("", "\t")

	p := ApkParser{
		zip:     zip,
		encoder: enc,
	}

	resourcesErr := p.parseResources()
	if resourcesErr != nil {
		return apkInfo, resourcesErr
	}
	manifestErr := p.ParseXml("AndroidManifest.xml")
	if manifestErr != nil {
		return apkInfo, manifestErr
	}

	var manifest Manifest
	_ = xml.Unmarshal(buf.Bytes(), &manifest)
	iconPath, _ := manifest.App.Icon.String()
	icon, err := p.ParseIcon(iconPath)
	if err != nil {
		return apkInfo, err
	}

	apkInfo.Package, _ = manifest.Package.String()
	apkInfo.VersionName, _ = manifest.VersionName.String()
	apkInfo.VersionCode, _ = manifest.VersionCode.Int32()
	apkInfo.Label, _ = manifest.App.Label.String()
	apkInfo.Icon = icon

	return apkInfo, nil
}

// Prepare the ApkParser instance, load resources if possible.
// encoder expects an XML encoder instance, like Encoder from encoding/xml package.
//
// This method will not Close() the zip, you are still the owner.
func NewParser(zip *ZipReader, encoder ManifestEncoder) (parser *ApkParser, resourcesErr error) {
	parser = &ApkParser{
		zip:     zip,
		encoder: encoder,
	}
	resourcesErr = parser.parseResources()
	return
}

func (p *ApkParser) parseResources() (err error) {
	if p.resources != nil {
		return nil
	}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Panic: %v\n%s", r, string(debug.Stack()))
		}
	}()

	resourcesFile := p.zip.File["resources.arsc"]
	if resourcesFile == nil {
		return os.ErrNotExist
	}

	if err := resourcesFile.Open(); err != nil {
		return fmt.Errorf("Failed to open resources.arsc: %s", err.Error())
	}
	defer resourcesFile.Close()

	p.resources, err = ParseResourceTable(resourcesFile)
	return
}

func (p *ApkParser) ParseXml(name string) error {
	file := p.zip.File[name]
	if file == nil {
		return fmt.Errorf("Failed to find %s in APK!", name)
	}

	if err := file.Open(); err != nil {
		return err
	}
	defer file.Close()

	var lastErr error
	for file.Next() {
		if err := ParseXml(file, p.encoder, p.resources); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	if lastErr == ErrPlainTextManifest {
		return lastErr
	}

	return fmt.Errorf("Failed to parse %s, last error: %v", name, lastErr)
}

func (p *ApkParser) ParseIcon(name string) (image.Image, error) {
	file := p.zip.File[name]
	if file == nil {
		return nil, fmt.Errorf("Failed to find %s in APK!", name)
	}

	if err := file.Open(); err != nil {
		return nil, err
	}
	defer file.Close()

	if strings.HasSuffix(file.Name, ".png") {
		icon, err := png.Decode(file)
		if err == nil {
			return icon, nil
		}
	}

	icon, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	} else {
		return icon, nil
	}
}
