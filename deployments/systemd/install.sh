#!/bin/bash
set -e

echo "=== Instalando Auth Service como servicio systemd ==="

# Verificar que se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Este script debe ejecutarse como root"
    exit 1
fi

# Variables
SERVICE_NAME="auth-service"
SERVICE_USER="auth-service"
SERVICE_GROUP="auth-service"
INSTALL_DIR="/opt/auth-service"
BINARY_PATH="../../bin/auth-service"

# Verificar que el binario existe
if [ ! -f "$BINARY_PATH" ]; then
    echo "Error: Binario no encontrado en $BINARY_PATH"
    echo "Ejecuta 'make build' primero"
    exit 1
fi

# Crear usuario y grupo si no existen
if ! id "$SERVICE_USER" &>/dev/null; then
    echo "Creando usuario $SERVICE_USER..."
    useradd --system --no-create-home --shell /bin/false "$SERVICE_USER"
fi

# Crear directorio de instalación
echo "Creando directorio de instalación..."
mkdir -p "$INSTALL_DIR"/{bin,logs,keys}

# Copiar binario
echo "Copiando binario..."
cp "$BINARY_PATH" "$INSTALL_DIR/bin/"
chmod +x "$INSTALL_DIR/bin/$SERVICE_NAME"

# Copiar claves RSA si existen
if [ -d "../../keys" ]; then
    echo "Copiando claves RSA..."
    cp ../../keys/*.pem "$INSTALL_DIR/keys/" 2>/dev/null || echo "Advertencia: No se encontraron claves RSA"
fi

# Copiar archivo .env si existe
if [ -f "../../.env" ]; then
    echo "Copiando archivo .env..."
    cp ../../.env "$INSTALL_DIR/"
else
    echo "Advertencia: No se encontró archivo .env"
    echo "Copia manualmente tu archivo .env a $INSTALL_DIR/"
fi

# Ajustar permisos
echo "Ajustando permisos..."
chown -R "$SERVICE_USER:$SERVICE_GROUP" "$INSTALL_DIR"
chmod 600 "$INSTALL_DIR/.env" 2>/dev/null || true
chmod 600 "$INSTALL_DIR/keys/"*.pem 2>/dev/null || true

# Instalar servicio systemd
echo "Instalando servicio systemd..."
cp auth-service.service /etc/systemd/system/
chmod 644 /etc/systemd/system/auth-service.service

# Recargar systemd
echo "Recargando systemd..."
systemctl daemon-reload

# Habilitar servicio
echo "Habilitando servicio..."
systemctl enable auth-service.service

echo ""
echo "=== Instalación completada ==="
echo ""
echo "Comandos útiles:"
echo "  Iniciar servicio:   sudo systemctl start auth-service"
echo "  Detener servicio:   sudo systemctl stop auth-service"
echo "  Ver estado:         sudo systemctl status auth-service"
echo "  Ver logs:           sudo journalctl -u auth-service -f"
echo "  Reiniciar servicio: sudo systemctl restart auth-service"
echo ""
echo "IMPORTANTE: Asegúrate de que el archivo .env esté configurado correctamente en:"
echo "  $INSTALL_DIR/.env"
echo ""
