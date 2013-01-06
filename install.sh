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
if [ -z $STARTUPSCRIPT ]; then
	STARTUPSCRIPT=grove.sh
fi
if [ -z $STARTUPSCRIPTLOC ]; then
	if [ ! -e "/etc/init.d" ]; then
		NOINITD=TRUE
	else
		STARTUPSCRIPTLOC=/etc/init.d/grove
	fi
fi

if [ "$1" = "--build" ] || [ "$1" = "-b" ]; then
	if [ "$(whoami)" = "root" ]; then
		echo "Some arrangements have difficulty building as root."
		echo "\033[1;35mIf the build fails, you may want to simply run"
		echo "  $COMPILER $COPTIONS && $0 \033[0m"
		echo
	fi
	
	echo "Building $GROVE..."
	$COMPILER
	
	echo " done."
fi

if [ ! -e "$GROVE" ]; then
	echo "The file '$GROVE' not found. Perhaps you should build first?"
	exit 1
fi

RESDIR=$(./$GROVE --show-res)
VERSION=$(./$GROVE --version)
echo "Resources directory: $RESDIR"

mkdir -p -m 755 $RESDIR
echo "Copying resources to $RESDIR"
cp -r res/* $RESDIR/

echo "Copying the $STARTUPSCRIPT startup script to $STARTUPSCRIPTLOC"
cp $STARTUPSCRIPT $STARTUPSCRIPTLOC
chmod +x $STARTUPSCRIPTLOC

if [ "$NOINITD" != FALSE ]; then
	echo "Moving $GROVE executable to $INSTALLDIR"
	chmod 755 $GROVE
	mv $GROVE $INSTALLDIR/
else
	echo "\033[1;31mPlease note:\033[1;0m"
	echo "/etc/init.d doesn't exist, so you're probably not running"
	echo "Debian or Ubuntu/Mint. As such, the startup script couldn't"
	echo "be copied to a proper location. You may want to move grove.sh"
	echo "to a place which is easy to access, so that you can start it"
	echo "up easily."
fi

echo "\033[1;32m### Installation finished. Version $VERSION\033[0m"
echo
echo "You can invoke Grove as follows:"
echo "  service $(basename $STARTUPSCRIPTLOC) start"
echo
echo "The path argument is generally either a specific repository, which"
echo "you might be looking to serve with Grove, or your entire development"
echo "directory, such as ~/dev or ~/src."
echo
echo "You can see the README for full instructions."