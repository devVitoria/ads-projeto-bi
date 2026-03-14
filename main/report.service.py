import csv
import json

jsonnn = {}
                
with open('./Fatura_2025-07-20.csv', newline='', encoding='UTF-8') as csvfile:    
    spamreader = csv.DictReader(csvfile, delimiter=',', quotechar=';')
    # print('CDSVSSSVS FILLE', spamreader)
    for i in spamreader:
        # print("IIII", i)
        for idx, (c, v) in enumerate(i.items()):
            # print('rodou', c, v)

            if (c in jsonnn.keys()):
                o2 = [] 
                # o2 = o2.append(jsonnn[c])
                print("AAA", c, v, jsonnn[c], jsonnn)
                o2 = o2.append(v)
                jsonnn[c] = o2
            else: 
                a = []
                a.append(v)
                jsonnn[c] = a
           
                        
print("ave", jsonnn)         
       
      




