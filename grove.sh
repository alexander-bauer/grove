#!/bin/bash
### Startup Script for Grove ###
# To enable: (on debian)
# ln -s /etc/init.d/grove /path/to/this/script/grove.sh
#
# To use:
# sudo service grove {start|stop|restart|status|check}
#

if [ -z $GROVE ]; then
	GROVE=grove
fi

if [ -z $LOG ]; then
	LOG=/tmp/grove.log
fi

if [ -z $DEV ]; then
	if [ -e ~/dev ]; then
		DEV=~/dev/
	elif [ -e ~/development/ ]; then
		DEV=~/development/ ]
	elif [ -e ~/code/ ]; then
		DEV=~/code/
	fi
fi

PID=$(pidof -o %PPID $GROVE)

start()
{
	if [ -z $PID ]; then
		if [ ! -z $(which $GROVE) ]; then
			$GROVE $DEV >> $LOG &
			echo "Started $GROVE"
			exit 0
		fi
		echo "$GROVE not found."
		exit 1
	fi
}

stop()
{
	if [ ! -z $PID ]; then
		echo "Killing '$GROVE', PID $PID"
		kill $PID
	fi
}

restart()
{
	stop
	start
}

status()
{
	echo -n "* Grove is "
	if [ -z $PID ]; then
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
		exit 0
	fi
	echo "Grove was running."
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
	"status" )
		status
		;;
	"check" )
		check
		;;
	* )
		echo "usage: $0 {start|stop|restart|status|check}"
esac