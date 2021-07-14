#!/bin/bash

#Java Setup
setup_java(){
  mkdir java
  tar zxf openjdk/OpenJDK11U-jdk_x64_linux_hotspot_11.0.11_9.tar.gz -C java --strip-components=4
  export JAVA_HOME=${BOSH_INSTALL_TARGET}/java
  export PATH=$JAVA_HOME/bin:$PATH
}

#Maven Setup
setup_maven(){
  mkdir maven-install
  tar zxf maven/apache-maven-3.6.3-bin.tar.gz -C maven-install --strip-components=1
  cp -R maven-install/* ${BOSH_INSTALL_TARGET}
  export M2_HOME=${BOSH_INSTALL_TARGET}/maven
  export PATH=$M2_HOME/bin:$PATH
}

#Cleanup Java files from BOSH_COMPILE_TARGET
cleanup_java(){
  rm -rf java
  rm -rf openjdk
}

#Cleanup Maven files from BOSH_COMPILE_TARGET
cleanup_maven(){
  rm -rf maven
}

