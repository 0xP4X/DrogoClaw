from PIL import Image
import sys

# Braille Unicode offset is 0x2800
# The 8 dots in a braille character are mapped to bits in this order:
# 1 4
# 2 5
# 3 6
# 7 8
pixel_map = [
    [0x01, 0x08],
    [0x02, 0x10],
    [0x04, 0x20],
    [0x40, 0x80]
]

def image_to_braille(image_path, width=40):
    try:
        img = Image.open(image_path).convert('RGBA')
    except Exception as e:
        print(e)
        return

    # To get `width` braille characters, we need 2*width pixels
    # To get `height` braille characters, we need 4*height pixels
    pixel_width = width * 2
    aspect_ratio = img.height / img.width
    pixel_height = int(pixel_width * aspect_ratio)
    
    # Ensure height is a multiple of 4
    pixel_height = pixel_height + (4 - pixel_height % 4) if pixel_height % 4 != 0 else pixel_height

    img = img.resize((pixel_width, pixel_height))
    
    # Create white background to handle transparency
    bg = Image.new('RGBA', img.size, (255, 255, 255))
    img = Image.alpha_composite(bg, img)
    img = img.convert('L') # grayscale
    
    pixels = img.load()
    
    braille_chars = []
    
    # Loop over 4x2 grids
    for y in range(0, pixel_height, 4):
        line = []
        for x in range(0, pixel_width, 2):
            char_val = 0
            for dy in range(4):
                for dx in range(2):
                    # If pixel is dark (less than threshold), set the bit
                    if pixels[x+dx, y+dy] < 200:
                        char_val |= pixel_map[dy][dx]
            
            if char_val == 0:
                line.append(" ") # Empty space instead of empty braille (0x2800) for cleaner copy-pasting
            else:
                line.append(chr(0x2800 + char_val))
        braille_chars.append("".join(line))
        
    print("\n".join(braille_chars))

image_to_braille("assets/logo.png", width=30)
