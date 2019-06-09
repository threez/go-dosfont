# go-dosfont

Implementation of the PSF/MZ font file format. With this library
you can use the old dos bitmap fonts in go.

# Example

	fonts, err := OpenFonts("fonts/NEW1252.FON")

    // use font face fonts[i].Face
    // of type golang.org/x/image/font/basicfont.Face

## Rendering

![fonts/NEW1252.FON.png](fonts/NEW1252.FON.png)

# TODO

* Implementation of the PE format

# Resources

Other resources for implementing the font lib.

## Implementations / Documentation

* https://www.seasip.info/Unix/PSF/
* https://www.lowlevel.eu/wiki/
* https://www.win.tue.nl/~aeb/linux/kbd/font-formats-1.html
* Microsoft_Portable_Executable_and_Common_Object_File_Format
* https://de.wikipedia.org/wiki/MZ-Datei
* http://www.delorie.com/djgpp/doc/exe/
* https://www.chiark.greenend.org.uk/~sgtatham/fonts/
* https://github.com/golang/image/blob/master/font/inconsolata/regular8x16.go
* https://fontforge.github.io/en-US/

## Test fonts

The fonts used for testing are sourced from https://www.uwe-sieber.de/dosfon.html. Thanks for your free font Uwe Sieber.
