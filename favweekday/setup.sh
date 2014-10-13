#!/bin/bash
mkdir -p github.com/gorilla
cd github.com/gorilla
git clone https://github.com/gorilla/context.git
git clone https://github.com/gorilla/mux.git
git clone https://github.com/gorilla/securecookie.git
git clone https://github.com/gorilla/sessions.git
cd -

mkdir -p github.com/google
cd github.com/google
git clone https://github.com/google/identity-toolkit-go-client.git
cd -

mkdir -p code.google.com/p
cd code.google.com/p
hg clone https://code.google.com/p/xsrftoken/
cd -
