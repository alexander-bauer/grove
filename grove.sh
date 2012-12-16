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
	LOG=grove.log
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
		$GROVE $DEV &>> $LOG &
	fi
}
stop()
{
	if [ ! -z $PID ]; then
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
		exit 1
	else
		echo "running."
		exit 0
	fi
}
check()
{
	status > /dev/null
	if [ $? == 1 ]; then
		start
	fi
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