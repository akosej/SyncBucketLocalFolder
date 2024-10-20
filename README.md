# agaSyncBucket

**agaSyncBucket** es una herramienta de sincronización de archivos que permite cargar automáticamente archivos desde una carpeta local a un bucket de MinIO. Esta aplicación facilita la gestión de archivos mediante un sistema de vigilancia que detecta cambios en la carpeta y actualiza el bucket de MinIO en consecuencia.

## Características

- Sincronización automática de archivos locales con un bucket de MinIO.
- Soporte para la creación y eliminación de archivos en el bucket.
- Configuración fácil a través de un archivo YAML.
- Instalación como comando del sistema para acceso rápido.
- Monitoreo de cambios en la carpeta local.

## Requisitos

- [Go](https://golang.org/doc/install) (versión 1.16 o superior)
- MinIO Server (puede ser instalado localmente o utilizar un servidor remoto)

## Instalación

1. **Clonar el repositorio:**
   ```bash
   git clone https://github.com/akosej/agaSyncBucket.git
   cd agaSyncBucket
   ```
2. Construir el proyecto:
   ```bash
    go build -o agaSyncBucket
   ```
3. Ejecutar el programa
   ```bash
   ./agaSyncBucket --install
   ```
4. Configurar el archivo de configuración: El archivo de configuración se creará automáticamente en `~/.minio_sync/config.yaml`. Edita este archivo para añadir tu configuración de MinIO:
   ```yml
    endPoint: tu_minio_endpoint
    accessKey: tu_access_key
    secretKey: tu_secret_key
    bucket: tu_bucket
    folder: tu_carpeta_local
    ssl: true  # o false si no usas SSL
   ```
## USO
Una vez que el programa esté instalado y configurado, simplemente ejecuta:
```bash
    ./agaSyncBucket
```

El programa comenzará a subir archivos existentes de la carpeta local especificada al bucket de MinIO. Además, se mantendrá en ejecución y monitoreará cualquier cambio en la carpeta local.

Opciones de línea de comandos

    --endPoint: Dirección del servidor MinIO.
    --accessKey: Clave de acceso para el servidor MinIO.
    --secretKey: Clave secreta para el servidor MinIO.
    --bucket: Nombre del bucket en MinIO.
    --folder: Ruta de la carpeta local que deseas sincronizar.
    --ssl: Usar SSL para la conexión (valor por defecto: false).
    --install: Instalar el programa como un comando del sistema.

## Ejemplo de uso
```bash
    ./agaSyncBucket --endPoint localhost:9000 --accessKey minioadmin --secretKey minioadmin --bucket mybucket --folder /ruta/a/mi/carpeta --ssl false
```