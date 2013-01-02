#!/bin/sh
set -e

if [ -z $COMPILER ]; then
	COMPILER="go build"
fi
if [ -z $COPTIONS ]; then
	COPTIONS=
fi
if [ -z $GROVE ]; then
	GROVE=grove
fi
if [ -z $INSTALLDIR ]; then
	INSTALLDIR=/usr/bin
fi

if [ "$1" != "skipbuild" ]; then
	if [ "$(whoami)" = "root" ]; then
		echo "Some arrangements have difficulty building as root."
		echo "If the build fails, you may want to simply run"
		echo "  $COMPILER $COPTIONS && $0 skipbuild"
		echo
	fi
	
	echo -n "Building $GROVE..."
	$COMPILER
	
	echo " done."
fi

RESDIR=$(./$GROVE --show-res)
VERSION=$(./$GROVE --version)
echo "Resources directory: $RESDIR"

mkdir -p -m 755 $RESDIR
echo "Copying resources to $RESDIR"

cp -f res/* $RESDIR/

echo "Moving $GROVE executable to $INSTALLDIR"
chmod 755 $GROVE
mv -f $GROVE $INSTALLDIR/

echo "\033[1;32mInstallation finished. Version $VERSION\033[0m"
echo "You can invoke Grove as follows."
echo "  $GROVE /path/to/serve"
echo "The path argument is generally either a specific repository, which"
echo "you might be looking to serve with Grove, or your entire development"
echo "directory, such as ~/dev or ~/src."
echo "You can see the README for full instructions."