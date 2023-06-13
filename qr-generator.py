import qrcode 

def generate_qr_code(url):
    img = qrcode.make(url)
    type(img)
    img.save("Pull-Request-URL.png") #save the image with the name "Pull-Request-URL.png"
url = input("Enter the url: ")
generate_qr_code(url)