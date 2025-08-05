package main

import (
	"fmt"
	"os"
	"net"
	"bytes"
	"encoding/binary"
	"log"
	"io"
)

const(
	FLAG_SYNC = 1 << 1 
	FLAG_ACK	= 1 << 4
)


type Packet struct {
    Seq   uint32
    Ack   uint32
    Flags byte
    Data  []byte
}

func createPacket(seq uint32, ack uint32, flags byte, payload []byte) []byte  {

	buf := new(bytes.Buffer)
	
	binary.Write(buf, binary.BigEndian, seq)
	binary.Write(buf, binary.BigEndian, ack)

	buf.WriteByte(flags)
	buf.Write(payload)

	return buf.Bytes()
}

func DeserializePacket(buf []byte) Packet {
    seq := binary.BigEndian.Uint32(buf[0:4])
    ack := binary.BigEndian.Uint32(buf[4:8])
    flags := buf[8]
    data := buf[9:]

    return Packet{
        Seq:   seq,
        Ack:   ack,
        Flags: flags,
        Data:  data,
    }
}

func main()  {
	addr := net.UDPAddr{
		Port:9000,
		IP: net.ParseIP("127.0.0.1"),
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	fmt.Println("Server escuchando en el puerto 9000")
	buffer := make([]byte, 1024)

  for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		fmt.Println("recibido", n)
		if err != nil {
			fmt.Println("Error al leer:", err)
			continue
		}

		msg := string(buffer[:n])
		fmt.Println("Cliente pidio:", msg)

file, err := os.Open("NoMoreTears.pcm")
if err != nil {
	log.Fatal(err)
}
defer file.Close()
pcmData, err := io.ReadAll(file)
if err != nil {
	log.Fatal(err)
}
//decoder, err := mp3.NewDecoder(f)
//fmt.Println(decoder.SampleRate())

if err != nil {
	log.Fatal(err)
}

//pcmData, err := io.ReadAll(decoder)
if err != nil {
	log.Fatal(err)
}


		seq := 0
		for i := 0; i < len(pcmData); i += 1024{
			end := i + 1024
			if end > len(pcmData){
				end = len(pcmData)
			}
      chunk := createPacket(uint32(seq), 0, 0, pcmData[i:end])
			conn.WriteToUDP(chunk, clientAddr)
			fmt.Println("Enviando",  i, " , ", end)
			seq += 1024

	    //esperar ack
			fmt.Println("esperando ACK")
			ackBuf := make([]byte, 1024)
			conn.Read(ackBuf)
			ackPacket := DeserializePacket(ackBuf)

			if ackPacket.Ack == 0 {
				fmt.Println("ack no llegó, cortando conexion")
				break
			}
			fmt.Println("ack crrecto", ackPacket.Ack)
			
		}

   conn.WriteToUDP([]byte("END"), clientAddr)
   fmt.Println("Enviado!")
	}
	fmt.Println("termino")
}
