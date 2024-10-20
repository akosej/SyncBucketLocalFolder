package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/akosej/SyncBucketLocalFolder/system"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/pterm/pterm"
	"github.com/radovskyb/watcher"
	"github.com/schollz/progressbar/v3"
	"gopkg.in/yaml.v2"
)

var (
	endPoint  = flag.String("endPoint", "", "Matrix homeserver")
	accessKey = flag.String("accessKey", "", "Access Key")
	secretKey = flag.String("secretKey", "", "Secret Key")
	bucket    = flag.String("bucket", "", "Name Bucket")
	folder    = flag.String("folder", "", "Path Local Folder")
	useSSL    = flag.Bool("ssl", false, "Use SSL")
	install   = flag.Bool("install", false, "Install as a system command")
)

// Configuración del programa
type Config struct {
	EndPoint  string `yaml:"endPoint"`
	AccessKey string `yaml:"accessKey"`
	SecretKey string `yaml:"secretKey"`
	Bucket    string `yaml:"bucket"`
	Folder    string `yaml:"folder"`
	SSL       bool   `yaml:"ssl"`
}

// Cargar la configuración desde el archivo
func loadConfig() (*Config, error) {
	usr, err := user.Current()
	if err != nil {
		return nil, err
	}

	configFile := filepath.Join(usr.HomeDir, ".minio_sync", "config.yaml")
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("el archivo de configuración no existe")
	}

	var config Config
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Copiar el archivo de configuración a la carpeta del usuario
func setupConfig() {
	usr, err := user.Current()
	if err != nil {
		pterm.Error.Println("Error al obtener el usuario actual:", err)
		os.Exit(1)
	}

	configDir := filepath.Join(usr.HomeDir, ".minio_sync")
	configFile := filepath.Join(configDir, "config.yaml")

	// Crear la carpeta si no existe
	if err := os.MkdirAll(configDir, 0755); err != nil {
		pterm.Error.Println("Error al crear el directorio de configuración:", err)
		os.Exit(1)
	}

	// Copiar el archivo de configuración de ejemplo si no existe
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		defaultConfig := fmt.Sprintf("endPoint: %s\naccessKey: %s\nsecretKey: %s\nbucket: %s\nfolder: %s\nssl: %v\n",
			system.EndPoint, system.AccessKey, system.SecretKey, system.Bucket, system.LocalFolder, system.UseSSL)
		if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
			pterm.Error.Println("Error al crear el archivo de configuración:", err)
			os.Exit(1)
		}
		pterm.Success.Println("Archivo de configuración creado en:", configFile)
	} else {
		pterm.Info.Println("El archivo de configuración ya existe en:", configFile)
	}
}

// Instalar el programa como un comando del sistema
func installCommand() {
	var cmdPath string
	var err error

	switch runtime.GOOS {
	case "windows":
		cmdPath, err = filepath.Abs(os.Args[0])
		if err != nil {
			pterm.Error.Println("Error al obtener la ruta del ejecutable:", err)
			os.Exit(1)
		}
		installScript := fmt.Sprintf("setx PATH \"%%PATH%%;%s\"", filepath.Dir(cmdPath))
		if err := os.WriteFile("install.bat", []byte(installScript), 0644); err != nil {
			pterm.Error.Println("Error al crear el archivo de instalación:", err)
			os.Exit(1)
		}
		pterm.Success.Println("Instalador creado: ejecuta install.bat para agregar el programa al PATH")
	default: // Linux, macOS, etc.
		cmdPath, err = filepath.Abs(os.Args[0])
		if err != nil {
			pterm.Error.Println("Error al obtener la ruta del ejecutable:", err)
			os.Exit(1)
		}
		installScript := fmt.Sprintf("export PATH=$PATH:%s", filepath.Dir(cmdPath))
		shellConfig := filepath.Join(os.Getenv("HOME"), ".bashrc")
		if runtime.GOOS == "darwin" {
			shellConfig = filepath.Join(os.Getenv("HOME"), ".zshrc")
		}

		// Agregar la línea de exportación al archivo de configuración del shell
		file, err := os.OpenFile(shellConfig, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			pterm.Error.Println("Error al abrir el archivo de configuración del shell:", err)
			os.Exit(1)
		}
		defer file.Close()

		if _, err := file.WriteString("\n" + installScript + "\n"); err != nil {
			pterm.Error.Println("Error al escribir en el archivo de configuración del shell:", err)
			os.Exit(1)
		}

		pterm.Success.Println("El programa ha sido instalado. Ejecuta 'source", shellConfig, "' para aplicar los cambios.")
	}
}

func main() {
	flag.Parse()

	if *install {
		setupConfig()
		installCommand()
		return
	}

	// Cargar configuración desde el archivo si existe
	config, err := loadConfig()
	if err != nil {
		pterm.Error.Println("Error al cargar la configuración:", err)
		os.Exit(1)
	}

	// Asignar valores desde los flags o desde el archivo de configuración si son nulos
	if *endPoint == "" {
		*endPoint = config.EndPoint
	}
	if *accessKey == "" {
		*accessKey = config.AccessKey
	}
	if *secretKey == "" {
		*secretKey = config.SecretKey
	}
	if *folder == "" {
		*folder = config.Folder
	}
	if *bucket == "" {
		*bucket = config.Bucket
	}
	if !*useSSL {
		*useSSL = config.SSL
	}

	if *endPoint == "" ||
		*accessKey == "" ||
		*secretKey == "" ||
		*folder == "" ||
		*bucket == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Inicializar el cliente MinIO
	minioClient, err := minio.New(*endPoint, &minio.Options{
		Creds:  credentials.NewStaticV4(*accessKey, *secretKey, ""),
		Secure: system.UseSSL,
	})
	if err != nil {
		pterm.Error.Println("Error al inicializar el cliente MinIO:", err)
		os.Exit(1)
	}

	// Asegurarse de que el bucket existe
	ctx := context.Background()
	exists, errBucketExists := minioClient.BucketExists(ctx, *bucket)
	if errBucketExists != nil {
		pterm.Error.Println("Error al verificar la existencia del bucket:", errBucketExists)
		os.Exit(1)
	}
	if !exists {
		err := minioClient.MakeBucket(ctx, *bucket, minio.MakeBucketOptions{})
		if err != nil {
			pterm.Error.Println("Error al crear el bucket:", err)
			os.Exit(1)
		}
		pterm.Success.Println("Bucket", *bucket, "creado correctamente")
	} else {
		pterm.Info.Println("Bucket", *bucket, "ya existe")
	}

	// Función para subir archivos al bucket con barra de progreso
	uploadFile := func(path string) {
		info, err := os.Stat(path)
		if err != nil {
			pterm.Error.Println("Error al obtener información del archivo:", err)
			return
		}
		if info.IsDir() {
			return
		}

		objectName := strings.TrimPrefix(path, *folder+string(os.PathSeparator))

		// Abrir el archivo para leerlo
		file, err := os.Open(path)
		if err != nil {
			pterm.Error.Println("Error al abrir archivo:", err)
			return
		}
		defer file.Close()

		// Crear la barra de progreso
		bar := progressbar.DefaultBytes(
			info.Size(),
			fmt.Sprintf("Subiendo %s", objectName),
		)

		// Subir el archivo usando io.MultiWriter para actualizar la barra de progreso
		_, err = minioClient.PutObject(ctx, *bucket, objectName, io.TeeReader(file, bar), info.Size(), minio.PutObjectOptions{})
		if err != nil {
			pterm.Error.Println("Error al subir archivo:", err)
		} else {
			pterm.Success.Println("Archivo subido:", objectName)
		}
	}

	// Función para eliminar archivos del bucket
	deleteFile := func(path string) {
		objectName := strings.TrimPrefix(path, *folder+string(os.PathSeparator))
		err := minioClient.RemoveObject(ctx, *bucket, objectName, minio.RemoveObjectOptions{})
		if err != nil {
			pterm.Error.Println("Error al eliminar archivo del bucket:", err)
		} else {
			pterm.Warning.Println("Archivo eliminado:", objectName)
		}
	}

	// Subir los archivos existentes en la carpeta al bucket
	pterm.Info.Println("Subiendo archivos existentes...")
	err = filepath.Walk(*folder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			uploadFile(path)
		}
		return nil
	})
	if err != nil {

		pterm.Error.Println("Error al listar los archivos:", err)
		os.Exit(1)
	}

	// Configurar el watcher
	w := watcher.New()
	w.SetMaxEvents(0)
	w.FilterOps(watcher.Create, watcher.Write, watcher.Rename, watcher.Move, watcher.Remove)

	// Agregar la carpeta al watcher
	if err := w.AddRecursive(*folder); err != nil {
		pterm.Error.Println("Error al agregar carpeta al watcher:", err)
		os.Exit(1)
	}

	// Iniciar la vigilancia en una gorutina
	go func() {
		for {
			select {
			case event := <-w.Event:
				switch event.Op {
				case watcher.Create, watcher.Write, watcher.Rename, watcher.Move:
					pterm.Info.Println("Cambio detectado en:", event.Path)
					uploadFile(event.Path)
				case watcher.Remove:
					pterm.Warning.Println("Archivo eliminado:", event.Path)
					deleteFile(event.Path)
				}
			case err := <-w.Error:
				pterm.Error.Println("Error:", err)
			case <-w.Closed:
				return
			}
		}
	}()

	// Iniciar el watcher
	pterm.Info.Println("Monitoreando cambios en la carpeta local:", *folder)
	if err := w.Start(time.Millisecond * 500); err != nil {
		pterm.Error.Println("Error al iniciar el watcher:", err)
		os.Exit(1)
	}
}
