#!/usr/bin/env bash
# prime_cache.sh - Downloads popular Go packages to seed the GolangZakhireh cache.

set -euo pipefail

# ANSI Colors for better visibility
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
GRAY='\033[1;30m'
NC='\033[0m'

# Configuration
SERVER_URL="http://localhost:8811/proxy"
export GOPROXY="$SERVER_URL,direct"
export GOSUMDB=off
export GOTOOLCHAIN=local

# List of popular/common Go packages
# This is a curated list of top-tier libraries across various categories
PACKAGES=(
    # Web Frameworks & Routers
    "github.com/gin-gonic/gin"
    "github.com/labstack/echo/v4@v4.13.3"
    "github.com/gofiber/fiber/v2"
    "github.com/gorilla/mux"
    "github.com/go-chi/chi/v5"
    "github.com/go-playground/validator/v10"
    "github.com/unrolled/secure"
    "github.com/justinas/alice"
    "github.com/gorilla/handlers"
    "github.com/gorilla/websocket"

    # Databases & ORMs
    "gorm.io/gorm"
    "gorm.io/driver/postgres"
    "gorm.io/driver/mysql"
    "gorm.io/driver/sqlite"
    "github.com/go-redis/redis/v8"
    "github.com/jackc/pgx/v4"
    "go.mongodb.org/mongo-driver/mongo"
    "github.com/volatiletech/sqlboiler/v4@v4.18.0"
    "github.com/upper/db/v4"
    "github.com/knadh/koanf"
    "github.com/dgraph-io/badger/v3"
    "github.com/blevesearch/bleve/v2"

    # Logging & CLI
    "github.com/sirupsen/logrus"
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
    "go.uber.org/zap"
    "github.com/rs/zerolog"
    "github.com/alecthomas/log4go"
    "github.com/mitchellh/mapstructure"
    "github.com/urfave/cli/v2"

    # Testing & Benchmarking
    "github.com/stretchr/testify"
    "github.com/onsi/ginkgo/v2@v2.22.0"
    "github.com/onsi/gomega"
    "github.com/DATA-DOG/go-sqlmock"
    "gotest.tools/v3/assert"
    "golang.org/x/tools/go/analysis"
    "github.com/vektra/mockery/v2"

    # Utilities
    "github.com/google/uuid"
    "github.com/pkg/errors"
    "golang.org/x/crypto"
    "golang.org/x/sys"
    "golang.org/x/net"
    "golang.org/x/sync"
    "golang.org/x/text"
    "google.golang.org/protobuf"
    "google.golang.org/grpc"
    "github.com/joho/godotenv"
    "github.com/google/go-cmp/cmp"
    "github.com/spf13/afero"
    "github.com/otiai10/copy"
    "github.com/hashicorp/go-multierror"
    "github.com/davecgh/go-spew/spew"
    "github.com/robfig/cron/v3"

    # Cloud & AWS
    "github.com/aws/aws-sdk-go-v2"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "cloud.google.com/go/storage"
    "github.com/linode/linodego@v1.44.0"
    "github.com/digitalocean/godo"

    # Serialization
    "github.com/json-iterator/go"
    "github.com/mailru/easyjson"
    "gopkg.in/yaml.v3"
    "github.com/pelletier/go-toml"
    "github.com/mitchellh/hashstructure/v2"
    "github.com/golang/protobuf/jsonpb"

    # Security & Misc
    "github.com/projectdiscovery/nuclei/v3@v3.3.0"
    "github.com/projectdiscovery/gologger@v1.1.28"
    "github.com/projectdiscovery/utils@v0.4.0"
    "golang.org/x/oauth2"
    "github.com/golang-jwt/jwt/v5"
    "github.com/ory/dockertest/v3"
    "github.com/securego/gosec/v2@v2.22.0"
    "github.com/caddyserver/certmagic@v0.22.0"

    # Networking & HTTP
    "github.com/hashicorp/consul/api"
    "github.com/gorilla/websocket"
    "github.com/valyala/fasthttp"
    "github.com/tidwall/gjson"
    "github.com/tidwall/sjson"
    "github.com/gorilla/schema"
    "github.com/julienschmidt/httprouter"
    "github.com/andybalholm/brotli"
    "github.com/gobwas/ws"

    # Observability & Metrics
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/sdk/trace"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
    "github.com/prometheus/client_golang/prometheus/promhttp"
    "github.com/uber/jaeger-client-go"
    "github.com/segmentio/kafka-go"

    # File & Image Handling
    "github.com/disintegration/imaging"
    "github.com/nfnt/resize"
    "github.com/h2non/filetype"
    "github.com/gabriel-vasile/mimetype"

    # Other Useful Packages
    "github.com/gookit/color"
    "github.com/cheggaaa/pb/v3"
    "github.com/thoas/go-funk"
    "github.com/dustin/go-humanize"
    "github.com/mattn/go-isatty"
    "github.com/mattn/go-colorable"
    "github.com/mattn/go-shellwords"
)

echo -e "Starting GolangZakhireh Cache Priming..."
echo
echo -e "  Proxy: ${GRAY}${SERVER_URL}${NC}"

# Create a temporary directory to avoid cluttering current dir
TEMP_DIR=$(mktemp -d)
trap "rm -rf $TEMP_DIR" EXIT
cd "$TEMP_DIR"
go mod init prime-cache >/dev/null 2>&1
echo -e "  Temporary module path: ${GRAY}$TEMP_DIR${NC}"
echo

count=0
success_count=0
failed_count=0
total=${#PACKAGES[@]}

for pkg in "${PACKAGES[@]}"; do
    ((count++))

    echo "──────────────────────────────────────────────────────"
    echo

    printf "[%d/%d] Downloading %b%s%b\n" \
        "$count" "$total" "$CYAN" "$pkg" "$NC"

    if output=$(go get "$pkg" 2>&1); then
        echo -e "  ${CYAN}→ Completed${NC}"
        ((success_count++))
    else
        echo -e "  ${RED}→  Failed${NC}"
        echo "     $output" | sed 's/^/     /'
        ((failed_count++))
    fi

    echo
done

echo "──────────────────────────────────────────────────────"
echo
echo -e " Cache priming finished!"
echo

# Calculate success percentage
percent=0
if [ $total -gt 0 ]; then
    percent=$(( success_count * 100 / total ))
fi

echo -e " ${BLUE}Summary:${NC}"
echo -e "   Total:     $total"
echo -e "   Succeeded: ${GREEN}$success_count${NC}"
echo -e "   Failed:    ${RED}$failed_count${NC}"
echo -e "   Success:   ${CYAN}$percent%${NC}"
echo
echo "Check your GolangZakhireh dashboard to see the cached modules."
echo
