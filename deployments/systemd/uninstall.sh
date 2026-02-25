#!/bin/bash
set -e

echo "=== Desinstalando Auth Service ==="

# Verificar que se ejecuta como root
if [ "$EUID" -ne 0 ]; then
    echo "Error: Este script debe ejecutarse como root"
    exit 1
fi

SERVICE_NAME="auth-service"
INSTALL_DIR="/opt/auth-service"

# Detener el servicio si está corriendo
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "Deteniendo servicio..."
    systemctl stop "$SERVICE_NAME"
fi

# Deshabilitar el servicio
if systemctl is-enabled --quiet "$SERVICE_NAME"; then
    echo "Deshabilitando servicio..."
    systemctl disable "$SERVICE_NAME"
fi

# Eliminar archivo de servicio
if [ -f "/etc/systemd/system/$SERVICE_NAME.service" ]; then
    echo "Eliminando archivo de servicio..."
    rm -f "/etc/systemd/system/$SERVICE_NAME.service"
fi

# Recargar systemd
echo "Recargando systemd..."
systemctl daemon-reload
systemctl reset-failed

# Preguntar si eliminar archivos de instalación
read -p "¿Deseas eliminar el directorio de instalación $INSTALL_DIR? (s/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Ss]$ ]]; then
    echo "Eliminando directorio de instalación..."
    rm -rf "$INSTALL_DIR"
    echo "Directorio eliminado"
else
    echo "Directorio de instalación conservado en: $INSTALL_DIR"
fi

# Preguntar si eliminar usuario
read -p "¿Deseas eliminar el usuario auth-service? (s/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Ss]$ ]]; then
    if id "auth-service" &>/dev/null; then
        echo "Eliminando usuario auth-service..."
        userdel auth-service
        echo "Usuario eliminado"
    fi
else
    echo "Usuario auth-service conservado"
fi

echo ""
echo "=== Desinstalación completada ==="
