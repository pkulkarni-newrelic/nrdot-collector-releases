#!/bin/bash

REPO_DIR="$( cd "$(dirname "$( dirname "${BASH_SOURCE[0]}" )")" &> /dev/null && pwd )"
BUILDER=''

# default values
skipcompilation=true
validate=true
fips=false
cgo=0

while getopts d:s:l:b:f: flag

do
    case "${flag}" in
        d) distributions=${OPTARG};;
        s) skipcompilation=${OPTARG};;
        l) validate=${OPTARG};;
        b) BUILDER=${OPTARG};;
        f) fips=${OPTARG};;
        *) exit 1;;
    esac
done

[[ -n "$BUILDER" ]] || BUILDER='ocb'

if [[ -z $distributions ]]; then
    echo "List of distributions to build not provided. Use '-d' to specify the names of the distributions to build. Ex.:"
    echo "$0 -d nrdot-collector-k8s"
    exit 1
fi

if [[ "$skipcompilation" = true ]]; then
    echo "Skipping the compilation, we'll only generate the sources."
elif [[ "$fips" == true ]]; then
    echo "❌ ERROR: FIPS requires skip compilation."
    echo "Skip Compilation is false."
    exit 1
fi

echo "Distributions to build: $distributions";

for distribution in $(echo "$distributions" | tr "," "\n")
do
    pushd "${REPO_DIR}/distributions/${distribution}" > /dev/null || exit

    manifest_file="manifest.yaml";
    build_folder="_build"

    if [[ "$fips" == true ]]; then
      yq eval '
         .dist.name += "-fips" |
         .dist.description += "-fips" |
         .dist.output_path += "-fips"' manifest.yaml > manifest-fips.yaml
      manifest_file="manifest-fips.yaml"
      build_folder="_build-fips"
      cgo=1
    fi

    # Enable CGO for distributions that require it (Oracle and SQL Server receivers)
    if [[ "$distribution" == "nrdot-collector-host" ]] || [[ "$distribution" == "nrdot-collector" ]]; then
      cgo=1
      
      # Check if C compiler is available for CGO
      if [[ "$skipcompilation" = false ]] && ! command -v gcc &> /dev/null && ! command -v clang &> /dev/null; then
        echo "❌ ERROR: CGO requires a C compiler (gcc or clang) but none was found."
        echo "   Install build tools:"
        echo "   - On Ubuntu/Debian: sudo apt-get install build-essential"
        echo "   - On RHEL/CentOS: sudo yum groupinstall 'Development Tools'"
        echo "   - On Alpine: apk add gcc musl-dev"
        exit 1
      fi
      
      # Check if Oracle Instant Client headers are available
      if [[ -z "$CGO_CFLAGS" ]] || [[ -z "$CGO_LDFLAGS" ]]; then
        echo "⚠️  WARNING: CGO_CFLAGS and CGO_LDFLAGS are not set."
        echo "   Oracle receiver requires Oracle Instant Client."
        echo "   Please set:"
        echo "   CGO_CFLAGS=\"-I/path/to/instantclient/sdk/include\""
        echo "   CGO_LDFLAGS=\"-L/path/to/instantclient -lclntsh\""
        echo "   LD_LIBRARY_PATH=/path/to/instantclient:\$LD_LIBRARY_PATH"
      fi
      
      # Export CGO flags if they're set in the environment so they're available to the builder
      if [[ -n "$CGO_CFLAGS" ]]; then
        export CGO_CFLAGS
      fi
      if [[ -n "$CGO_LDFLAGS" ]]; then
        export CGO_LDFLAGS
      fi
      if [[ -n "$LD_LIBRARY_PATH" ]]; then
        export LD_LIBRARY_PATH
      fi
    fi

    mkdir -p $build_folder

    echo "Building: $distribution"
    echo "Using Builder: $(command -v "$BUILDER")"
    echo "Using Go: $(command -v go)"
    echo "Using FIPS: ${fips}"
    echo "Using CGO_ENABLED: ${cgo}"
    
    # Export CGO_ENABLED so it's available to the builder and its subprocesses
    export CGO_ENABLED=${cgo}
    
    # Debug: Show what CGO variables are set
    if [[ ${cgo} -eq 1 ]]; then
        echo "CGO_CFLAGS: ${CGO_CFLAGS}"
        echo "CGO_LDFLAGS: ${CGO_LDFLAGS}"
        echo "LD_LIBRARY_PATH: ${LD_LIBRARY_PATH}"
    fi

    # For CGO builds, we need to generate sources first, then compile manually
    # because the builder doesn't properly pass through CGO environment variables
    if [[ ${cgo} -eq 1 ]] && [[ "$skipcompilation" = false ]]; then
        echo "Generating sources with builder (CGO build requires two-step process)..."
        if "$BUILDER" --skip-compilation=true --config ${manifest_file} > ${build_folder}/build.log 2>&1; then
            echo "Sources generated successfully. Now compiling with CGO..."
            pushd ${build_folder} > /dev/null || exit
            
            # Read the binary name from manifest
            binary_name=$(grep "name:" ../${manifest_file} | head -1 | awk '{print $2}')
            
            if CGO_ENABLED=${cgo} go build -trimpath -o "${binary_name}" -ldflags="-s -w" . >> build.log 2>&1; then
                echo "✅ SUCCESS: distribution '${distribution}' built with CGO."
            else
                echo "❌ ERROR: failed to compile '${distribution}' with CGO."
                echo "🪵 Build logs for '${distribution}'"
                echo "----------------------"
                cat build.log
                echo "----------------------"
                popd > /dev/null || exit
                exit 1
            fi
            popd > /dev/null || exit
        else
            echo "❌ ERROR: failed to generate sources for '${distribution}'."
            echo "🪵 Build logs for '${distribution}'"
            echo "----------------------"
            cat $build_folder/build.log
            echo "----------------------"
            exit 1
        fi
    else
        # Standard build path for non-CGO builds
        if "$BUILDER" --skip-compilation="${skipcompilation}" --config ${manifest_file} > ${build_folder}/build.log 2>&1; then
            if [[ "$fips" == true ]]; then
                echo "Copying fips.go into _build-fips."
                cp ../../fips/fips.go ./$build_folder
            fi
            echo "✅ SUCCESS: distribution '${distribution}' built."
        else
            echo "❌ ERROR: failed to build the distribution '${distribution}'."
            echo "🪵 Build logs for '${distribution}'"
            echo "----------------------"
            cat $build_folder/build.log
            echo "----------------------"
            exit 1
        fi
    fi

    popd > /dev/null || exit
done
