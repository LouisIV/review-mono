import struct, zlib, shutil

def create_rgba_png(size):
    raw = b''
    for y in range(size):
        raw += b'\x00'  # filter: none
        for x in range(size):
            raw += b'\x3B\x82\xF6\xFF'  # RGBA blue
    
    def chunk(ctype, data):
        c = ctype + data
        return struct.pack('>I', len(data)) + c + struct.pack('>I', zlib.crc32(c) & 0xFFFFFFFF)
    
    ihdr = struct.pack('>IIBBBBB', size, size, 8, 6, 0, 0, 0)
    return b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', ihdr) + chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b'')

for s in [32, 128]:
    with open(f'{s}x{s}.png', 'wb') as f:
        f.write(create_rgba_png(s))

shutil.copy('128x128.png', 'icon.icns')
shutil.copy('32x32.png', 'icon.ico')
print('done')
