Abyssal blackbox recorder
==
 
This application is used as a telemetry recorder for Abyssal space runs in EVE Online. Analyzer available at https://abyssal.space
 
Goal of this approach is to provide something similar to _Warcraftlogs_, just for EVE Online Abyssal journeys!
 
There is [Abyss Tracker](https://abyss.eve-nt.uk/), why you created this? Capturing overview and combatlog allows a lot more detailed analysis and overview of run. I don't see this project as competition for Abyss Tracker, it's just a totally different approach. I won't mind even sending your run details (loots, timing and etc) to Abyss Tracker so that you could keep both sites in sync.
 
How does it work?
--
 
* Application captures area of your overview (configured inside application) every second and stored as frames of animated GIF image.
* Combat log of characters selected also captured (from moment you click `Start Recording` until `Stop recording` is clicked)
* Application also listens to you `Clipboard`, this is for easy loot capture. (listens for change in clipboard and records it)
 
Additionally combatlog language is detected (as a hint for an analytics engine).
All above mentioned data is gzip-compressed and written to the file when `Stop Recording` button is pressed.
How/what data stored you can find in files `*.proto` in the `protobuf` folder.
 
For your convenience there is additional executable `extract.exe` included to uncompress `.abyss` files for inspection.
 
When you visit https://abyssal.space site and login with your EVE account, you can upload `.abyss` files for analytics (this is currently an early preview, a lot more additional data points will be available later).
 
TL;DR; version:
* It captures frames as your streaming applications would (1 frame per second)
* Captures logs written to files (only characters selected) while recording
* Captures clipboard changes (CTRL+A - select all, CTRL+C - copy) in your inventory window for easy loot recording.
 
When uploaded, overview capture is processed as animated sequence and timings/spawns are detected using OCR (Optical character recognition, and other algorithms).
 
For recording to be accurate, `Abyssal Trace` MUST be on your overview, that's the trigger for discovering when you enter and exit abyssal space.
 
It doesn't touch EVE Online client in any way. Application uses Windows API to enumerate EVE client windows by Title of the window, and later uses Windows API to capture region of window (what you configure).
 
How to setup for recording
--
 
In general settings:
* Set fontsize to `Medium`
* Transparency to `0`
* Select `Dark Matter` theme (for test drive, later you can play with another one)
 
![General Settings](/screenshots/general.png)
 
Overview
--
 
![Overview](/screenshots/overview.png)
 
In overview make sure following is setup:
* `Abyssal Trace` is in your overview (appears in overview when you activate Abyssal filament) (**A MUST!**)
* `Type` column is at least 256 pixels wide and sorter accordingly (triangle up, this to make sure that `Vila Swarmer` drones won't push your regular NPC out of overview)
* In Overview settings > Appearance make sure `use small font` is NOT checked!
* Loot boxes in on your overview: (currently not used, but later could be used to detect looting)
    * `Triglavian Extraction Node`
    * `Triglavian Extraction SubNode`
    * `Triglavian Bioadaptive Cache`
    * `Triglavian Biocombinative Cache`
* Wrecks is in overview too:
    * `Extraction Node Wreck`
    * `Extraction SubNode Wreck`
    * `Biocombinative Cache Wreck`
    * `Triglavian Bioadaptive Cache Wreck`
* Gates are on overview:
    * `Origin Conduit (Triglavian)`
    * `Transfer Conduit (Triglavian)`
* Remove friendly state ships from overview (in general, keep it only limited to stuff found in Abyssal space).
 
All above mentioned settings allow for more detailed analysis to be performed.
 
Good example of how capture area should look like:

![Captured region example](/screenshots/capture.png)

Notice that no other columns of overview is in play!

Recording (from scratch)
--
* Launch EVE Clients, login, prepare everything you need
* Launch recorder, configure it to grab `Type` column of your overview
* Make sure combatlog directory is correctly discovered, if not, browse for it and checkbox character that gonna run it.
* Click `Start Record`
* In your inventory window, press CTRL+A, CTRL+C (this copies your initial inventory)
* Activate Abyssal Filament
* When Fillament disappears from your inventory (consumed), again , press CTRL+A, CTRL+C (this will capture your inventory after removal of filament. This allows of perfect recognition what type of Abyssal site you are running)
* Go do the Abyss site, while looting you can occasionally record your loot with CTRL+A, CTRL+C
* When you have some spare time, check what weather strength is and press shortcut to set weather strength in recording. 
* After you exit abyss, or looted last can and hit reload (important if tracking of ammo used is important to you) record your loot again.
* Click `Stop Recording`, this will write timestamped file inside `recordings` folder where application executable is.

Later video guide will be available explaining everything in more details.
 
Is this allowed by CCP?
--
 
This repository open-sourced for this specific reason - to get CCP eyes on the code and whitelist/approve.

After creating ticket with support of EVE online I was told by GM that CCP specifically doesn't approve/whitelist any third party software.

It safe to use if it doesn't brake their EULA and other policies, and I'm sure it's not, we are not interacting with EVE client in anyway, just fetching part of window from Windows API. As we know already by many other programs reading of combat log is allowed. We don't touch cache or other stuff, you can read code yourself.
  
How to contact
-
You can find me in Abyssal Lurkers Discord (ShiVAs#5949) or `#abyssal-telemetry` channel, or send email: lurker at abyssal dot space.
 

