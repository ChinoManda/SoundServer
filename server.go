package main

import (
	"fmt"
	"os"
	"net"
	"bytes"
	"encoding/binary"
	"log"
	"io"
	"strings"
	"time"
)


const (
	FlagACK    = 1 << 0 // 00000001
	FlagSYNC   = 1 << 1 // 00000010
	FlagAUDIO  = 1 << 2 // 00000100
	FlagSTOP   = 1 << 3 // 00001000
	FlagMETA   = 1 << 4 // 00010000
	FlagCONFIG = 1 << 5 // 00100000
	FlagCHOICE = 1 << 6 // 01000000


	maxRetries = 5

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

func handShake(conn *net.UDPConn, Packet Packet, clientAddr *net.UDPAddr) bool {
	seq := uint32(2000)
	clientAck := Packet.Ack
	handShakePacket := createPacket(seq, Packet.Ack+1, FlagSYNC | FlagACK, nil)
	conn.WriteToUDP(handShakePacket, clientAddr)
  buffer := make([]byte, 1024)
  conn.Read(buffer)
  response := DeserializePacket(buffer)
  if response.Seq == clientAck+1 && response.Ack == seq+1 {
  	return true
  }
	return false
}

func grabSong(song string) []byte {
	
song = strings.TrimRight(song, "\x00")
file, err := os.Open("music/" + song)
if err != nil {
	fmt.Println("error: ", song)
	log.Fatal(err)
}
defer file.Close()
pcmData, err := io.ReadAll(file)
if err != nil {
	log.Fatal(err)
}
return pcmData
}

func sendSong(pcmData []byte, conn *net.UDPConn, clientAddr *net.UDPAddr)  {
	
		seq := 0
		for i := 0; i < len(pcmData); i += 1024{
			end := i + 1024
			if end > len(pcmData){
				end = len(pcmData)
			}
			pcmDataChunk := pcmData[i:end]
      chunk := createPacket(uint32(seq), 0, FlagAUDIO, pcmDataChunk)
			conn.WriteToUDP(chunk, clientAddr)
			fmt.Println("Enviando",  i, " , ", end)
	    //esperar ack
			fmt.Println("esperando ACK")

			retries := 0 
			ackReceived := false 
			for retries < maxRetries{
				conn.SetReadDeadline(time.Now().Add(1 * time.Second))
			    ackBuf := make([]byte, 1024)
					_, err :=	conn.Read(ackBuf)
   	    if err != nil {
				// Timeout o error
				fmt.Println("ACK no recibido, reenviando paquete...")
				conn.WriteToUDP(chunk, clientAddr)
				retries++
				continue
				}

		    ackPacket := DeserializePacket(ackBuf)
				if ackPacket.Ack == uint32(seq)+uint32(len(pcmDataChunk)) {
				fmt.Println("ACK recibido correctamente:", ackPacket.Ack)
				ackReceived = true
				break
					} else {
						fmt.Println("ACK incorrecto:", ackPacket.Ack, "esperado:", uint32(seq)+uint32(len(pcmDataChunk)))
						conn.WriteToUDP(chunk, clientAddr)
						retries++
						}
			}
   
			seq += 1024
	if !ackReceived {
		fmt.Println("No se recibió ACK después de varios intentos, cerrando conexión.")
		break
		}
			
		}

   conn.WriteToUDP([]byte("END"), clientAddr)
   fmt.Println("Enviado!")
	
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
    Packet := DeserializePacket(buffer)
		
    switch  {
    case Packet.Flags&FlagCHOICE != 0 :
		fmt.Println("FlagCHOICE")
		pcmData := grabSong(string(Packet.Data))
		sendSong(pcmData, conn, clientAddr)
		fmt.Println("Tengo cancion nashe")
	  case Packet.Flags&FlagSYNC != 0 :
			fmt.Println("FlagSYNC")
			success := handShake(conn, Packet, clientAddr)	
   if success {
   	fmt.Println("handShake correcto!")
   } else {
		 fmt.Println("handShake incorrecto")
	 }
    }

 }
}


