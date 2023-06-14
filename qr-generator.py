import qrcode 

def generate_qr_code(url):
    img = qrcode.make(url)
    img.save("img2.png")

def get_comment(external_urls):
      body = '|  Name | Link |\n|---------------------------------|------------------------|'
      generate_qr_code(external_urls[0])
      body += "📱" + "Preview on mobile" + '![QR Code](img2.png)|'
      return body
external_urls = [input("Enter the url: ")]
print(get_comment(external_urls))