#!/bin/bash -ex

EXPORTED_ARTIFACTS=exported-artifacts
mkdir -p $EXPORTED_ARTIFACTS

make image-driver image-operator

