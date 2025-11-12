#!/bin/bash

# AI  - Playwright 
# bash scripts/setup-playwright.sh [output-dir]

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
PLAYWRIGHT_VERSION="1.49.1"  #  playwright-go v0.4902.0

# 
TARGET_OS="${2:-}"
TARGET_ARCH="${3:-}"

#  bin/playwright
OUTPUT_DIR="${1:-$ROOT_DIR/bin/playwright}"
NODE_DIR="$OUTPUT_DIR/node"
BROWSER_DIR="$OUTPUT_DIR/browsers"

echo " Playwright..."
echo "$OUTPUT_DIR"

# 
HOST_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
echo ": $HOST_OS"

if [ -n "$TARGET_OS" ] && [ "$TARGET_OS" != "$HOST_OS" ]; then
    echo " $TARGET_OS  Playwright$HOST_OS"
    echo " Node.js "
    echo ""
fi

# 
mkdir -p "$OUTPUT_DIR"
mkdir -p "$NODE_DIR"
mkdir -p "$BROWSER_DIR"

# 1.  Node.js 
if [ ! -f "$NODE_DIR/bin/node" ]; then
    echo "[1/5]  Node.js..."

    # 
    if [ -n "$TARGET_OS" ] && [ -n "$TARGET_ARCH" ]; then
        OS="$TARGET_OS"
        ARCH="$TARGET_ARCH"
        echo ": $OS/$ARCH"
    else
        OS=$(uname -s | tr '[:upper:]' '[:lower:]')
        ARCH=$(uname -m)
        echo ": $OS/$ARCH"
    fi

    case "$OS" in
        linux)
            NODE_OS="linux"
            ;;
        darwin)
            NODE_OS="darwin"
            ;;
        msys*|mingw*|cygwin*)
            NODE_OS="win"
            ;;
        *)
            echo " : $OS"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64)
            NODE_ARCH="x64"
            ;;
        arm64|aarch64)
            NODE_ARCH="arm64"
            ;;
        *)
            echo " : $ARCH"
            exit 1
            ;;
    esac

    NODE_VERSION="20.11.0"
    NODE_FILENAME="node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}"

    if [ "$NODE_OS" = "win" ]; then
        NODE_FILENAME="${NODE_FILENAME}.zip"
    else
        NODE_FILENAME="${NODE_FILENAME}.tar.gz"
    fi

    #  Node.js
    cd /tmp
    echo "   : https://nodejs.org/dist/v${NODE_VERSION}/${NODE_FILENAME}"
    if curl -L -o "$NODE_FILENAME" "https://nodejs.org/dist/v${NODE_VERSION}/${NODE_FILENAME}" 2>/dev/null; then
        echo "   "
    else
        echo " "
        rm -f "$NODE_FILENAME"
        exit 1
    fi

    # 
    echo "    $NODE_FILENAME..."
    if [ "$NODE_OS" = "win" ]; then
        if command -v unzip >/dev/null 2>&1; then
            unzip -q "$NODE_FILENAME" && mv "node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}" node-tmp
        else
            echo "  unzip  Windows Node.js "
            rm -f "$NODE_FILENAME"
            exit 1
        fi
    else
        tar -xzf "$NODE_FILENAME" && mv "node-v${NODE_VERSION}-${NODE_OS}-${NODE_ARCH}" node-tmp
    fi

    if [ $? -ne 0 ]; then
        echo " "
        rm -rf node-tmp
        rm -f "$NODE_FILENAME"
        exit 1
    fi

    # 
    if [ -d "node-tmp" ]; then
        mv node-tmp/* "$NODE_DIR/" 2>/dev/null || cp -R node-tmp/* "$NODE_DIR/"
        rm -rf node-tmp
        rm -f "$NODE_FILENAME"
        echo " Node.js "
    else
        echo " : "
        rm -f "$NODE_FILENAME"
        exit 1
    fi
else
    echo " Node.js "
fi

# 2.  Playwright NPM 
echo "[2/4]  Playwright NPM ..."

cd "$OUTPUT_DIR"

#  package.json
if [ ! -f "package.json" ]; then
    "$NODE_DIR/bin/npm" init -y
fi

#  Playwright CLI
"$NODE_DIR/bin/npm" install --save-dev @playwright/test@"$PLAYWRIGHT_VERSION"

# 3. 
echo "[3/4]  Playwright ..."

# 
export PLAYWRIGHT_BROWSERS_PATH="$BROWSER_DIR"

#  Chromium
"$NODE_DIR/bin/npx" playwright install chromium

echo " "

# 4.  Playwright  playwright-go
echo "[4/5]  Playwright Go ..."

#  playwright-go 
DRIVER_SOURCE_DIR="${OUTPUT_DIR}/node_modules/playwright"
DRIVER_TARGET_DIR="${OUTPUT_DIR}/playwright-driver"

if [ -d "$DRIVER_SOURCE_DIR" ]; then
    cp -R "$DRIVER_SOURCE_DIR" "$DRIVER_TARGET_DIR"
    echo " : ${OUTPUT_DIR}/playwright-driver"
else
    echo " :  playwright : $DRIVER_SOURCE_DIR"
    exit 1
fi

# 4.1 
echo "[4.5/5] ..."
MISSING_FILES=0

if [ ! -f "$DRIVER_TARGET_DIR/package.json" ]; then
    echo " :  package.json"
    MISSING_FILES=$((MISSING_FILES + 1))
fi

if [ ! -d "$DRIVER_TARGET_DIR/lib" ]; then
    echo " :  lib/ "
    MISSING_FILES=$((MISSING_FILES + 1))
fi

BROWSER_COUNT=$(ls -1 "$BROWSER_DIR" 2>/dev/null | wc -l)
if [ "$BROWSER_COUNT" -lt 2 ]; then
    echo " :  ( $BROWSER_COUNT )"
else
    echo "  $BROWSER_COUNT "
fi

if [ $MISSING_FILES -eq 0 ]; then
    echo " "
else
    echo "  $MISSING_FILES "
fi

# 4.2 
echo "[4.6/5] ..."
NPM_VERSION=$(cd "$OUTPUT_DIR" && "$NODE_DIR/bin/npm" list @playwright/test 2>/dev/null | grep @playwright/test | sed 's/.*@//' | head -1)
DRIVER_VERSION=$(grep '"version"' "$DRIVER_TARGET_DIR/package.json" 2>/dev/null | sed 's/.*\"\(.*\)\".*/\1/')

if [ "$NPM_VERSION" = "$DRIVER_VERSION" ]; then
    echo " : $NPM_VERSION"
else
    echo " : npm=$NPM_VERSION, driver=$DRIVER_VERSION"
fi

# 5. 
echo "[5/5] ..."

cat > "$OUTPUT_DIR/playwright-path.sh" << 'EOF'
#!/bin/bash
#  Playwright 
DIR="$(cd "$(dirname "$0")" && pwd)"
export PLAYWRIGHT_NODE_HOME="$DIR/node"
export PLAYWRIGHT_BROWSERS_PATH="$DIR/browsers"
export PLAYWRIGHT_DRIVER_PATH="$DIR/playwright-driver"
export PATH="$DIR/node/bin:$PATH"
EOF

chmod +x "$OUTPUT_DIR/playwright-path.sh"

echo ""
echo "Playwright "
echo ""
echo ""
echo "1.  Playwright: source $OUTPUT_DIR/playwright-path.sh && npx playwright --version"
echo "2. : ls -lh $BROWSER_DIR"
echo "3. : ls -lh $OUTPUT_DIR/playwright-driver/package.json"
echo "4. : cd $ROOT_DIR/backend && go run -exec 'env PLAYWRIGHT_DRIVER_PATH=$OUTPUT_DIR/playwright-driver' ."
echo ""
echo ""
echo "  node/           Node.js runtime"
echo "  node_modules/   NPM packages"
echo "  browsers/       Playwright browsers"
echo "  playwright-driver/  Go driver for playwright-go"
echo ""
echo ""
echo "  PLAYWRIGHT_DRIVER_PATH=$OUTPUT_DIR/playwright-driver"
echo "  PLAYWRIGHT_BROWSERS_PATH=$BROWSER_DIR"
echo ""
