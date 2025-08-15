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
	FlagSONGS  = 1 << 7 // 10000000

	maxRetries = 5
  PacketSize = 4096
)

type Packet struct {
    Seq   uint32
    Ack   uint32
    Flags byte
    Data  []byte
}

type Client struct {
	Conn *net.UDPConn
	ClientAddr *net.UDPAddr
	Ch 			chan []byte
  AckCh 	chan Packet
}

var clients = make(map[string]*Client)

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
	fmt.Println("Bb")
  conn.Read(buffer)
  response := DeserializePacket(buffer)
	fmt.Println(response.Seq, response.Ack, clientAck, seq)
  if response.Seq == clientAck+1 && response.Ack == seq+1 {
  	return true
  }
	return false
}

func grabSong(song string) []byte {
	
song = strings.TrimRight(song, "\x00")
file, err := os.Open("music/" + song + ".pcm")
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

func sendSong(pcmData []byte, client *Client)  {
	
		seq := 0
		bytesEnviados := 0
		for i := 0; i < len(pcmData); i += PacketSize{
			end := i + PacketSize
			if end > len(pcmData){
				end = len(pcmData)
			}
    
			pcmDataChunk := pcmData[i:end]
      chunk := createPacket(uint32(seq), 0, FlagAUDIO, pcmDataChunk)
			client.Conn.WriteToUDP(chunk, client.ClientAddr)
			fmt.Println("Enviando",  i, " , ", end)
			bytesEnviados += PacketSize
			fmt.Println("Porcentaje: ", float64(bytesEnviados) / float64(len(pcmData)) * 100)
	    //esperar ack
			retries := 0 
			ackReceived := false 
			for retries < maxRetries{
      select {
			case ack := <-client.AckCh:
				fmt.Println("Escuchamos y juzgamos")
				if ack.Ack == uint32(seq)+uint32(len(pcmDataChunk)) {
				ackReceived = true
				fmt.Println("Trueeeeeeeee")
				break 
					} else {
						fmt.Println("ACK incorrecto:", ack.Ack, "esperado:", uint32(seq)+uint32(len(pcmDataChunk)))
						client.Conn.WriteToUDP(chunk, client.ClientAddr)
						retries++
						}

		  case <-time.After(4 * time.Second):
              fmt.Println("Timeout, reenviando paquete...", retries)
						client.Conn.WriteToUDP(chunk, client.ClientAddr)
              retries++
            
			}

	if ackReceived {
		break
		}
		}
	seq += PacketSize
	if !ackReceived {
		fmt.Println("No se recibió ACK después de varios intentos, cerrando conexión.")
		break
		}		
 }
		PacketEnd := createPacket(0,0,FlagSTOP, nil)
   client.Conn.WriteToUDP(PacketEnd, client.ClientAddr)
   fmt.Println("Terminado!")	
}
func sendSongNOAck(pcmData []byte, conn *net.UDPConn, clientAddr *net.UDPAddr) {
	
		seq := 0
		bytesEnviados := 0
		for i := 0; i < len(pcmData); i += PacketSize{
			end := i + PacketSize
			if end > len(pcmData){
				end = len(pcmData)
			}
    
			pcmDataChunk := pcmData[i:end]
      chunk := createPacket(uint32(seq), 0, FlagAUDIO, pcmDataChunk)
			conn.WriteToUDP(chunk, clientAddr)
		//	fmt.Println("Enviando",  i, " , ", end)
			bytesEnviados += PacketSize
		//	fmt.Println("Porcentaje: ", float64(bytesEnviados) / float64(len(pcmData)) * 100)
		}
		PacketEnd := createPacket(0,0,FlagSTOP, nil)
   conn.WriteToUDP(PacketEnd, clientAddr)
   fmt.Println("Terminado!")
}
func main()  {
	addr := net.UDPAddr{
		Port:9000,
		IP: net.IPv4zero,
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

   key := clientAddr.String()
	 if client, exists := clients[key]; exists{
		  pkt := DeserializePacket(buffer[:n])
			if pkt.Flags&FlagACK != 0 && pkt.Seq == 0 {
				fmt.Println("CLiente existe y flag ack")
				client.AckCh <- pkt
			} else {
			chunk := make([]byte, n)
			copy(chunk, buffer[:n])
			client.Ch <- chunk
		  } 
	 } else {
			clients[key] = &Client{
            Conn: conn,
						ClientAddr: clientAddr,
            Ch:   make(chan []byte),
						AckCh: make(chan Packet),
        }
				fmt.Println("Cliente agregado")
        handleClient(clients[key])
				copyData := make([]byte, n)
				copy(copyData, buffer[:n])
				clients[key].Ch <- copyData
	 }
 }
}
func handleClient(client *Client)  {
	go func ()  {
    defer func() {
        if r := recover(); r != nil {
            fmt.Println("Recovered in client handler:", r)
        }
    }()
		for data := range client.Ch{
		fmt.Println("clienteee")
     Pkt := DeserializePacket(data)
    fmt.Println(Pkt.Flags)
    switch  {
    case Pkt.Flags&FlagCHOICE != 0 :
		fmt.Println("FlagCHOICE")
		pcmData := grabSong(string(Pkt.Data))
		sendSong(pcmData, client)
	  case Pkt.Flags&FlagSYNC != 0 :
			fmt.Println("FlagSYNC")
			success := handShake(client.Conn, Pkt, client.ClientAddr)	
			fmt.Println(success)
  		 if success {
   			fmt.Println("handShake correcto!")
  			 } else {
				 fmt.Println("handShake incorrecto")
				 }
	  case Pkt.Flags&FlagSONGS != 0 :
			fmt.Println("FlagSONGS")
			sendChoices(client.Conn, client.ClientAddr)
    default :
			fmt.Println("defaulteo")
    } 
	  }
	}()
		
	}
	

func sendChoices(conn *net.UDPConn, clientAddr *net.UDPAddr){
	  entries, err := os.ReadDir("./music/")
    if err != nil {
        log.Fatal(err)
    }
 
    for _, e := range entries {
    song, _, _ := strings.Cut(e.Name(), ".")
		PacketSong := createPacket(0,0,FlagSONGS, []byte(song))
		_, err := conn.WriteToUDP(PacketSong, clientAddr)
			if err != nil {
		panic(err)
	}
    }
		PacketEND := createPacket(0,0,FlagSTOP,nil)
		conn.WriteToUDP(PacketEND, clientAddr)

}


