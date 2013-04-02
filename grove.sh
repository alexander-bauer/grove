#!/bin/bash
### BEGIN INIT INFO
# Provides:          grove
# Required-Start:    $remote_fs $syslog
# Required-Stop:     $remote_fs $syslog
# Default-Start:     none
# Default-Stop:      0 1 2 3 4 5 6
# Short-Description: Service script for the Grove daemon.
# Description:       Start, stop, or restart the Grove webserver/daemon.
### END INIT INFO
#
# To use:
# sudo service grove {start|stop|restart|status|check}
#

# Determine the full path of the grove binary, if not set
if [ -z $GROVE ]; then
	GROVE=$(which grove)
fi

# Attempt to locate the directory from which to serve files if not set
if [ -z "$SRC" ]; then
	if [ -e ~/src ]; then
		SRC=~/src
	elif [ -e ~/dev ]; then
		SRC=~/dev
	elif [ $(git status >> /dev/null; echo $?) == 0 ]; then
		# If the working directory is a git repository, use that. 
		SRC=$(pwd)
	fi
fi

# Set default options if not set
if [ -z "$OPTIONS" ]; then
	OPTIONS=""
fi

# Set the location to log to, if not set
if [ -z $LOG ]; then
	LOG=/dev/null
fi

# Find the PID of any running instance owned by the current user
PID=$(pgrep -u "$(whoami)" -f -d " " $GROVE)

start()
{
	if [ -z "$SRC" ]; then
		echo "Could not determine the directory to serve"
		return 1
	fi
	if [ ! -z "$PID" ]; then
		echo "Grove is already running."
		return 1
	fi
	if [ -z "$GROVE" ]; then
		echo "Grove not found."
		return 1
	fi

	$GROVE $OPTIONS $SRC >> $LOG &
	echo "Started $GROVE serving $SRC"
	return 0
}

stop()
{
	if [ ! -z "$PID" ]; then
		echo "Killing '$GROVE', PID $PID"
		kill $PID
	fi
}

restart()
{
	stop
	PID="" start
}

status()
{
	echo -n "* Grove is "
	if [ -z "$PID" ]; then
		echo "not running."
		return 1
	else
		echo "running."
		return 0
	fi
}

check()
{
	status > /dev/null
	if [ $? == 1 ]; then
		echo "Grove was not running."
		start
		return 0
	fi
	echo "Grove is running."
}

case "$1" in
	"start" )
		start
		;;
	"stop" )
		stop
		;;
	"restart" )
		restart
		;;
	"force-reload" )
		restart
		;;
	"status" )
		status
		;;
	"check" )
		check
		;;
	* )
		echo "usage: $0 {start|stop|restart|status|check}"
esac
