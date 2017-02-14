#!/usr/bin/env bash
REPO_NAME="terraform-provider-cloudfoundry"
NAME="terraform-provider-cloudfoundry"
OS=""
OWNER="orange-cloudfoundry"
: "${TMPDIR:=${TMP:-$(CDPATH=/var:/; cd -P tmp)}}"
cd -- "${TMPDIR:?NO TEMP DIRECTORY FOUND!}" || exit
cd -

which terraform &> /dev/null
if [[ "$?" != "0" ]]; then
    echo "you must have terraform installed"
fi
tf_version=$(terraform --version | awk '{print $2}')
tf_version=${tf_version:1:3}

if [[ "x$PROVIDER_CLOUDFOUNDRY_VERSION" == "x" ]]; then
    VERSION=$(curl -s https://api.github.com/repos/${OWNER}/${REPO_NAME}/releases/latest | grep tag_name | head -n 1 | cut -d '"' -f 4)
else
    VERSION=$PROVIDER_CLOUDFOUNDRY_VERSION
fi

echo "Installing ${NAME}-${VERSION}..."
if [[ "$OSTYPE" == "linux-gnu" || "$(uname -s)" == "Linux" ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
elif [[ "$OSTYPE" == "cygwin" ]]; then
    OS="windows"
elif [[ "$OSTYPE" == "msys" ]]; then
    OS="windows"
elif [[ "$OSTYPE" == "win32" ]]; then
    OS="windows"
else
    echo "Os not supported by install script"
    exit 1
fi

ARCHNUM=`getconf LONG_BIT`
ARCH=""
CPUINFO=`uname -m`
if [[ "$ARCHNUM" == "32" ]]; then
    ARCH="386"
else
    ARCH="amd64"
fi
if [[ "$CPUINFO" == "arm"* ]]; then
    ARCH="arm"
fi
FILENAME="${NAME}_${tf_version}_${OS}_${ARCH}"
if [[ "$OS" == "windows" ]]; then
    FILENAME="${FILENAME}.exe"
fi

LINK="https://github.com/${OWNER}/${REPO_NAME}/releases/download/${VERSION}/${FILENAME}"
if [[ "$OS" == "windows" ]]; then
    FILEOUTPUT="${FILENAME}"
else
    FILEOUTPUT="${TMPDIR}/${FILENAME}"
fi
RESPONSE=200
if hash curl 2>/dev/null; then
    RESPONSE=$(curl --write-out %{http_code} -L -o "${FILEOUTPUT}" "$LINK")
else
    wget -o "${FILEOUTPUT}" "$LINK"
    RESPONSE=$?
fi

if [ "$RESPONSE" != "200" ] && [ "$RESPONSE" != "0" ]; then
    echo "File ${LINK} not found, so it can't be downloaded."
    rm "$FILEOUTPUT"
    exit 1
fi

chmod +x "$FILEOUTPUT"
mkdir -p ~/.terraform.d/providers/
if [[ "$OS" == "windows" ]]; then
    mv "$FILEOUTPUT" "${HOME}/.terraform.d/providers/${NAME}"
else
    mv "$FILEOUTPUT" "${HOME}/.terraform.d/providers/${NAME}"
fi
provider_path="${HOME}/.terraform.d/providers/terraform-provider-cloudfoundry"
grep -Fxq "providers {" ~/.terraformrc &> /dev/null
if [[ $? != 0 ]]; then
    cat <<EOF >> ~/.terraformrc
providers {
    cloudfoundry = "$provider_path"
}
EOF
else
    grep -Fxq "cloudfoundry" ~/.terraformrc &> /dev/null
    if [[ $? != 0 ]]; then
        echo "${NAME}-${VERSION} has been installed."
        exit 0
    fi
    awk '/providers {/ { print; print "cloudfoundry = \"provider_path\""; next }1' ~/.terraformrc > /tmp/.terraformrc
    mv /tmp/.terraformrc ~/
fi

echo "${NAME}-${VERSION} has been installed."
