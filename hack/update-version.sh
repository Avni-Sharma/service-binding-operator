#!/bin/bash
# Update the operator version to a new version at various places across the repository.
# Refer https://semver.org/

MANIFESTS_DIR="./../manifests"
OPERATOR_VERSION="0.0.20"
version="${OPERATOR_VERSION}"
version=(${version//./$'\n'})  # change the semicolons to white space

OLD_VERSION_MAJOR="${version[0]}"
OLD_VERSION_MINOR="${version[1]}"
OLD_VERSION_PATCH="${version[2]}"
echo ${OLD_VERSION_MAJOR}
echo ${OLD_VERSION_MINOR}
echo ${OLD_VERSION_PATCH}
NEW_VERSION_MAJOR=$1
NEW_VERSION_MINOR=$2
NEW_VERSION_PATCH=$3

function replace {
    LOCATION=$1
    sed -i -e 's/'${OLD_VERSION_MAJOR}'.'${OLD_VERSION_MINOR}'.'${OLD_VERSION_PATCH}'/'${NEW_VERSION_MAJOR}'.'${NEW_VERSION_MINOR}'.'${NEW_VERSION_PATCH}'/g' $LOCATION
}
replace ../Makefile
replace ./update-version.sh
replace ${MANIFESTS_DIR}/service-binding-operator.package.yaml
replace ${MANIFESTS_DIR}/service-binding-operator.v${OLD_VERSION_MAJOR}.${OLD_VERSION_MINOR}.${OLD_VERSION_PATCH}.clusterserviceversion.yaml
mv ${MANIFESTS_DIR}/service-binding-operator.v${OLD_VERSION_MAJOR}.${OLD_VERSION_MINOR}.${OLD_VERSION_PATCH}.clusterserviceversion.yaml \
${MANIFESTS_DIR}/service-binding-operator.v${NEW_VERSION_MAJOR}.${NEW_VERSION_MINOR}.${NEW_VERSION_PATCH}.clusterserviceversion.yaml
replace ./../openshift-ci/Dockerfile.registry.build
echo -e "\n\033[0;32m \xE2\x9C\x94 Operator version upgraded from \
${OLD_VERSION_MAJOR}.${OLD_VERSION_MINOR}.${OLD_VERSION_PATCH} to \
${NEW_VERSION_MAJOR}.${NEW_VERSION_MINOR}.${NEW_VERSION_PATCH} \033[0m\n"