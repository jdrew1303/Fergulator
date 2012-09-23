package main

import (
	"fmt"
)

const (
	BankUpper = iota
	BankLower
)

type Mmc1 struct {
	RomBanks  [][]Word
	VromBanks [][]Word

	PrgBankCount  int
	ChrRomCount   int
	BatteryBacked bool
	Data          []byte

	Buffer        int
	BufferCounter uint
	PrgSwapBank   int
	PrgBankSize   int
	ChrBankSize   int
}

func (m *Mmc1) Write(v Word, a int) {
	// If reset bit is set
	if v&0x80 != 0 {
        fmt.Println("Resetting MMC")
		m.BufferCounter = 0
		m.Buffer = 0

        m.PrgSwapBank = BankLower
        m.PrgBankSize = Size16k
	} else {
		// Buffer the write
		m.Buffer = (m.Buffer & (0xFF - (0x1 << m.BufferCounter))) | ((int(v) & 0x1) << m.BufferCounter)
		m.BufferCounter++

		// If the buffer is filled
		if m.BufferCounter == 0x5 {
			m.SetRegister(m.RegisterNumber(a), m.Buffer)

			// Reset buffer
			m.BufferCounter = 0
			m.Buffer = 0
		}
	}
}

func (m *Mmc1) SetRegister(reg int, v int) {
	switch reg {
	// Control register
	case 0:
		fmt.Printf("Mapper: MMC1\n  Mirroring: ")
		// Set mirroring
		switch v & 0x3 {
		case 0x0:
			ppu.Nametables.SetMirroring(MirroringSingleUpper)
			fmt.Println("Single Upper")
		case 0x1:
			ppu.Nametables.SetMirroring(MirroringSingleLower)
			fmt.Println("Single Lower")
		case 0x2:
			ppu.Nametables.SetMirroring(MirroringVertical)
			fmt.Println("Vertical")
		case 0x3:
			ppu.Nametables.SetMirroring(MirroringHorizontal)
			fmt.Println("Horizontal")
		}

        switch (v >> 0x2) & 0x3 {
        case 0x0:
            fallthrough
        case 0x1:
            m.PrgBankSize = Size32k
			m.PrgSwapBank = BankLower
            fmt.Printf("  PRG Swap Bank: Lower\n")
            fmt.Printf("  PRG Bank Size: 32k\n")
        case 0x2:
            m.PrgBankSize = Size16k
			m.PrgSwapBank = BankUpper
            fmt.Printf("  PRG Swap Bank: Upper\n")
            fmt.Printf("  PRG Bank Size: 16k\n")
        case 0x3:
            m.PrgBankSize = Size16k
			m.PrgSwapBank = BankLower
            fmt.Printf("  PRG Swap Bank: Lower\n")
            fmt.Printf("  PRG Bank Size: 16k\n")
        }

		fmt.Printf("  CHR Bank Size: ")
		// Set CHR bank size
		switch (v >> 0x4) & 0x1 {
		case 0x0:
			m.ChrBankSize = Size8k
			fmt.Println("8k")
		case 0x1:
			m.ChrBankSize = Size4k
			fmt.Println("4k")
		}
		// CHR Bank 0
	case 1:
		if m.ChrRomCount == 0 {
			return
		}

		// Select VROM at 0x0000
		switch m.ChrBankSize {
		case Size8k:
			// Swap 8k VROM (in 8k mode, ignore first bit D0)
			bank := (v >> 0x1) & 0xF
			WriteVramBank(m.VromBanks, bank, 0x0, Size8k)
		case Size4k:
			// Swap 4k VROM
			bank := v & 0x1F
			WriteVramBank(m.VromBanks, bank, 0x0, Size4k)
		}
		// CHR Bank 1
	case 2:
		if m.ChrRomCount == 0 {
			return
		}

		// Select VROM bank at 0x1000, ignored in
		// 8k switching mode
		if m.ChrBankSize == Size4k {
			bank := v & 0x1F
			WriteRamBank(m.RomBanks, bank, 0x1000, Size4k)
		}
		// PRG Bank
	case 3:
		switch m.PrgBankSize {
		case Size32k:
			// Swap 32k ROM (in 32k mode, ignore first bit D0)
			bank := ((v >> 0x1) & 0x7) * 2
			fmt.Printf("32k write to: %d\n", bank/2)

			WriteRamBank(m.RomBanks, bank, 0x8000, Size16k)
			WriteRamBank(m.RomBanks, bank+1, 0xC000, Size16k)
		case Size16k:
			// Swap 16k ROM
			bank := v & 0xF

			if m.PrgSwapBank == BankUpper {
				WriteRamBank(m.RomBanks, bank, 0xC000, Size16k)
			} else {
				WriteRamBank(m.RomBanks, bank, 0x8000, Size16k)
			}
		}
	}
}

func (m *Mmc1) RegisterNumber(a int) int {
	switch {
	case a >= 0x8000 && a <= 0x9FFF:
		return 0
	case a >= 0xA000 && a <= 0xBFFF:
		return 1
	case a >= 0xC000 && a <= 0xDFFF:
		return 2
	}

	return 3
}