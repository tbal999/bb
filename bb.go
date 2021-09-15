package main

import "os"

/*
   Copyright (C) 2021 - tom@fern91.com

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.
   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.
   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

var (
	//THESE TWO VARS MUST BE CONFIGURED
	admin      = "tom" ////////////////////////// the username for administrator. As administrator you MUST run "./bb init" to ensure BB is set up correctly.
	boardtitle = "heathens.club bb"
)

func main() {
	initiateBB(os.Args)
}
