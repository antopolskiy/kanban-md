// Run by macOS's built-in JavaScript for Automation runtime, not Node.js.
ObjC.import('AppKit');

function present(value) {
    return value && !value.isNil();
}

function run(argv) {
    var board = $.NSPasteboard.generalPasteboard;
    if (argv[0] === 'image') {
        var png = board.dataForType($.NSPasteboardTypePNG);
        if (!present(png)) {
            var image = $.NSImage.alloc.initWithPasteboard(board);
            if (present(image)) {
                var bitmap = $.NSBitmapImageRep.imageRepWithData(image.TIFFRepresentation);
                if (!present(bitmap)) {
                    return JSON.stringify({kind: 'error'});
                }
                png = bitmap.representationUsingTypeProperties(
                    $.NSBitmapImageFileTypePNG, $.NSDictionary.dictionary);
                if (!present(png)) {
                    return JSON.stringify({kind: 'error'});
                }
            }
        }
        if (present(png)) {
            if (png.length > 20 * 1024 * 1024) {
                return JSON.stringify({kind: 'too-large'});
            }
            return JSON.stringify({kind: 'image', data: ObjC.unwrap(png.base64EncodedStringWithOptions(0))});
        }
    }
    var text = board.stringForType($.NSPasteboardTypeString);
    if (present(text)) {
        return JSON.stringify({kind: 'text', text: ObjC.unwrap(text)});
    }
    return JSON.stringify({kind: 'empty'});
}
