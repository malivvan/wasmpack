package main

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

func main() {
	var name string
	var silent bool
	var output string
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "usage %s <path>:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.BoolVar(&silent, "s", false, "silent mode (default: false)")
	flag.StringVar(&name, "n", "", "name of the global function (default: run immediately)")
	flag.StringVar(&output, "o", "", "output file (default: stdout)")
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	path := flag.Arg(0)

	var wasm []byte
	var err error
	if strings.HasSuffix(path, ".wasm") {
		wasm, err = os.ReadFile(path)
		if err != nil {
			println("read error: " + err.Error())
			os.Exit(1)
		}
	} else {
		wasm, err = build(path)
		if err != nil {
			println("build error: " + err.Error())
			os.Exit(1)
		}
	}
	code, err := pack(wasm)
	if err != nil {
		println("pack error: " + err.Error())
		os.Exit(1)
	}
	code, err = wrap(name, code)
	if err != nil {
		println("wrap error: " + err.Error())
		os.Exit(1)
	}

	if !silent {
		_, err = fmt.Fprintf(os.Stderr, "%s: %.2f MB -> %.2f MB (%.2f%%)\n", strings.TrimPrefix(output, "./"), float64(len(wasm))/1024/1024, float64(len(code))/1024/1024, float64(len(code))/float64(len(wasm))*100)
		if err != nil {
			println("info error: " + err.Error())
			os.Exit(1)
		}
	}

	var w io.Writer
	if output != "" {
		f, err := os.Create(output)
		if err != nil {
			println("create error: " + err.Error())
			os.Exit(1)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}
	_, err = w.Write([]byte(code))
	if err != nil {
		println("write error: " + err.Error())
		os.Exit(1)
	}

}

func pack(wasm []byte) (string, error) {
	buf := &bytes.Buffer{}
	w, err := flate.NewWriter(buf, flate.BestCompression)
	if err != nil {
		return "", fmt.Errorf("error creating deflate object: %w", err)
	}
	n, err := w.Write(wasm)
	if err != nil {
		return "", fmt.Errorf("error compressing data: %w", err)
	}
	if len(wasm) != n {
		return "", fmt.Errorf("buffer size mismatch: expected %d, got %d", len(wasm), n)
	}
	err = w.Flush()
	if err != nil {
		return "", fmt.Errorf("error flushing deflate object: %w", err)
	}
	err = w.Close()
	if err != nil {
		return "", fmt.Errorf("error closing deflate object: %w", err)
	}
	return `((input) => {
    	const ENCODING = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=",
        W_SIZE = 32768, STORED_BLOCK = 0, STATIC_TREES = 1, DYN_TREES = 2, L_BITS = 9, D_BITS = 6,
        MASK_BITS = [0x0000, 0x0001, 0x0003, 0x0007, 0x000f, 0x001f, 0x003f, 0x007f, 0x00ff, 0x01ff, 0x03ff, 0x07ff, 0x0fff, 0x1fff, 0x3fff, 0x7fff, 0xffff],
        COPY_LENS = [3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, 67, 83, 99, 115, 131, 163, 195, 227, 258, 0, 0],
        COPY_LEXT = [0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, 5, 5, 5, 5, 0, 99, 99],
        COPY_DIST = [1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, 769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577],
        COPY_DEXT = [0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, 11, 11, 12, 12, 13, 13],
        BORDER = [16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15];
    let slide, wp, fixed_tl, fixed_td, fixed_bl, fixed_bd, bit_buf, bit_len, method, eof, copy_leng, copy_dist, tl, td, bl, bd, inflate_data, inflate_pos;
    fixed_tl = null;
    let List = class {
        constructor() {
            this.next = null;
            this.list = null;
        }
    }
    let Node = class {
        constructor() {
            this.e = 0;
            this.b = 0;
            this.n = 0;
            this.t = null;
        }
    }
    let Build = class {
        constructor(b, n, s, d, e, mm) {
            this.BMAX = 16;
            this.N_MAX = 288;
            this.status = 0;
            this.root = null;
            this.m = 0;
            let a;
            let c = [];
            let el;
            let f;
            let g;
            let h;
            let i;
            let j;
            let k;
            let lx = [];
            let p;
            let pidx;
            let q;
            let r = new Node();
            let u = [];
            let v = [];
            let w;
            let x = [];
            let xp;
            let y;
            let z;
            let o;
            let tail;
            tail = this.root = null;
            for (i = 0; i < this.BMAX + 1; i++) c[i] = 0;
            for (i = 0; i < this.BMAX + 1; i++) lx[i] = 0;
            for (i = 0; i < this.BMAX; i++) u[i] = null;
            for (i = 0; i < this.N_MAX; i++) v[i] = 0;
            for (i = 0; i < this.BMAX + 1; i++) x[i] = 0;
            el = n > 256 ? b[256] : this.BMAX;
            p = b;
            pidx = 0;
            i = n;
            do {
                c[p[pidx]]++;
                pidx++;
            } while (--i > 0);
            if (c[0] === n) {
                this.root = null;
                this.m = 0;
                this.status = 0;
                return;
            }
            for (j = 1; j <= this.BMAX; j++) if (c[j] !== 0) break;
            k = j;
            if (mm < j) mm = j;
            for (i = this.BMAX; i !== 0; i--) if (c[i] !== 0) break;
            g = i;
            if (mm > i) mm = i;
            for (y = 1 << j; j < i; j++, y <<= 1) {
                if ((y -= c[j]) < 0) {
                    this.status = 2;
                    this.m = mm;
                    return;
                }
            }
            if ((y -= c[i]) < 0) {
                this.status = 2;
                this.m = mm;
                return;
            }
            c[i] += y;
            x[1] = j = 0;
            p = c;
            pidx = 1;
            xp = 2;
            while (--i > 0) x[xp++] = (j += p[pidx++]);
            p = b;
            pidx = 0;
            i = 0;
            do if ((j = p[pidx++]) !== 0) v[x[j]++] = i;
            while (++i < n);
            n = x[g];
            x[0] = i = 0;
            p = v;
            pidx = 0;
            h = -1;
            w = lx[0] = 0;
            q = null;
            z = 0;
            for (null; k <= g; k++) {
                a = c[k];
                while (a-- > 0) {
                    while (k > w + lx[1 + h]) {
                        w += lx[1 + h];
                        h++;
                        z = (z = g - w) > mm ? mm : z;
                        if ((f = 1 << (j = k - w)) > a + 1) {
                            f -= a + 1;
                            xp = k;
                            while (++j < z) {
                                if ((f <<= 1) <= c[++xp]) break;
                                f -= c[xp];
                            }
                        }
                        if (w + j > el && w < el) j = el - w;
                        z = 1 << j;
                        lx[1 + h] = j;
                        q = [];
                        for (o = 0; o < z; o++) q[o] = new Node();
                        if (!tail) tail = this.root = new List();
                        else tail = tail.next = new List();
                        tail.next = null;
                        tail.list = q;
                        u[h] = q;
                        if (h > 0) {
                            x[h] = i;
                            r.b = lx[h];
                            r.e = 16 + j;
                            r.t = q;
                            j = (i & ((1 << w) - 1)) >> (w - lx[h]);
                            u[h - 1][j].e = r.e;
                            u[h - 1][j].b = r.b;
                            u[h - 1][j].n = r.n;
                            u[h - 1][j].t = r.t;
                        }
                    }
                    r.b = k - w;
                    if (pidx >= n) {
                        r.e = 99;
                    } else if (p[pidx] < s) {
                        r.e = (p[pidx] < 256 ? 16 : 15);
                        r.n = p[pidx++];
                    } else {
                        r.e = e[p[pidx] - s];
                        r.n = d[p[pidx++] - s];
                    }
                    f = 1 << (k - w);
                    for (j = i >> w; j < z; j += f) {
                        q[j].e = r.e;
                        q[j].b = r.b;
                        q[j].n = r.n;
                        q[j].t = r.t;
                    }
                    for (j = 1 << (k - 1); (i & j) !== 0; j >>= 1) i ^= j;
                    i ^= j;
                    while ((i & ((1 << w) - 1)) !== x[h]) {
                        w -= lx[h];
                        h--;
                    }
                }
            }
            this.m = lx[1];
            this.status = ((y !== 0 && g !== 1) ? 1 : 0);
        }
    }
    let GET_BYTE = () => {
        if (inflate_data.length === inflate_pos) return -1;
        return inflate_data[inflate_pos++] & 0xff;
    }
    let NEED_BITS = (n) => {
        while (bit_len < n) {
            bit_buf |= GET_BYTE() << bit_len;
            bit_len += 8;
        }
    }
    let GET_BITS = (n) => bit_buf & MASK_BITS[n];
    let DUMP_BITS = (n) => {
        bit_buf >>= n;
        bit_len -= n;
    }
    let inflate_codes = (buff, off, size)=> {
        let e;
        let t;
        let n;
        if (size === 0) return 0;
        n = 0;
        for (;;) {
            NEED_BITS(bl);
            t = tl.list[GET_BITS(bl)];
            e = t.e;
            while (e > 16) {
                if (e === 99) return -1;
                DUMP_BITS(t.b);
                e -= 16;
                NEED_BITS(e);
                t = t.t[GET_BITS(e)];
                e = t.e;
            }
            DUMP_BITS(t.b);
            if (e === 16) {
                wp &= W_SIZE - 1;
                buff[off + n++] = slide[wp++] = t.n;
                if (n === size) return size;
                continue;
            }
            if (e === 15) break;
            NEED_BITS(e);
            copy_leng = t.n + GET_BITS(e);
            DUMP_BITS(e);
            NEED_BITS(bd);
            t = td.list[GET_BITS(bd)];
            e = t.e;
            while (e > 16) {
                if (e === 99) return -1;
                DUMP_BITS(t.b);
                e -= 16;
                NEED_BITS(e);
                t = t.t[GET_BITS(e)];
                e = t.e;
            }
            DUMP_BITS(t.b);
            NEED_BITS(e);
            copy_dist = wp - t.n - GET_BITS(e);
            DUMP_BITS(e);
            while (copy_leng > 0 && n < size) {
                copy_leng--;
                copy_dist &= W_SIZE - 1;
                wp &= W_SIZE - 1;
                buff[off + n++] = slide[wp++] = slide[copy_dist++];
            }
            if (n === size) return size;
        }
        method = -1;
        return n;
    }
    let inflate_stored = (buff, off, size) => {
        let n = bit_len & 7;
        DUMP_BITS(n);
        NEED_BITS(16);
        n = GET_BITS(16);
        DUMP_BITS(16);
        NEED_BITS(16);
        if (n !== ((~bit_buf) & 0xffff)) return -1;
        DUMP_BITS(16);
        copy_leng = n;
        n = 0;
        while (copy_leng > 0 && n < size) {
            copy_leng--;
            wp &= W_SIZE - 1;
            NEED_BITS(8);
            buff[off + n++] = slide[wp++] = GET_BITS(8);
            DUMP_BITS(8);
        }
        if (copy_leng === 0) method = -1;
        return n;
    }
    let inflate_fixed = (buff, off, size)=> {
        if (!fixed_tl) {
            let i;
            let l = [];
            let h;
            for (i = 0; i < 144; i++) l[i] = 8;
            for (null; i < 256; i++) l[i] = 9;
            for (null; i < 280; i++) l[i] = 7;
            for (null; i < 288; i++) l[i] = 8;
            fixed_bl = 7;
            h = new Build(l, 288, 257, COPY_LENS, COPY_LEXT, fixed_bl);
            if (h.status !== 0) throw new Error("HufBuild error: " + h.status);
            fixed_tl = h.root;
            fixed_bl = h.m;
            for (i = 0; i < 30; i++) l[i] = 5;
            fixed_bd = 5;
            h = new Build(l, 30, 0, COPY_DIST, COPY_DEXT, fixed_bd);
            if (h.status > 1) {
                fixed_tl = null;
                throw new Error("HufBuild error: " + h.status);
            }
            fixed_td = h.root;
            fixed_bd = h.m;
        }
        tl = fixed_tl;
        td = fixed_td;
        bl = fixed_bl;
        bd = fixed_bd;
        return inflate_codes(buff, off, size);
    }
    let inflate_dynamic = (buff, off, size) => {
        let i;
        let j;
        let l;
        let n;
        let t;
        let nb;
        let nl;
        let nd;
        let ll = [];
        let h;
        for (i = 0; i < 286 + 30; i++) ll[i] = 0;
        NEED_BITS(5);
        nl = 257 + GET_BITS(5);
        DUMP_BITS(5);
        NEED_BITS(5);
        nd = 1 + GET_BITS(5);
        DUMP_BITS(5);
        NEED_BITS(4);
        nb = 4 + GET_BITS(4);
        DUMP_BITS(4);
        if (nl > 286 || nd > 30) return -1;
        for (j = 0; j < nb; j++) {
            NEED_BITS(3);
            ll[BORDER[j]] = GET_BITS(3);
            DUMP_BITS(3);
        }
        for (null; j < 19; j++) ll[BORDER[j]] = 0;
        bl = 7;
        h = new Build(ll, 19, 19, null, null, bl);
        if (h.status !== 0) return -1;
        tl = h.root;
        bl = h.m;
        n = nl + nd;
        i = l = 0;
        while (i < n) {
            NEED_BITS(bl);
            t = tl.list[GET_BITS(bl)];
            j = t.b;
            DUMP_BITS(j);
            j = t.n;
            if (j < 16) {
                ll[i++] = l = j;
            } else if (j === 16) {
                NEED_BITS(2);
                j = 3 + GET_BITS(2);
                DUMP_BITS(2);
                if (i + j > n) return -1;
                while (j-- > 0) ll[i++] = l;
            } else if (j === 17) {
                NEED_BITS(3);
                j = 3 + GET_BITS(3);
                DUMP_BITS(3);
                if (i + j > n) return -1;
                while (j-- > 0) ll[i++] = 0;
                l = 0;
            } else {
                NEED_BITS(7);
                j = 11 + GET_BITS(7);
                DUMP_BITS(7);
                if (i + j > n) return -1;
                while (j-- > 0) ll[i++] = 0;
                l = 0;
            }
        }
        bl = L_BITS;
        h = new Build(ll, nl, 257, COPY_LENS, COPY_LEXT, bl);
        if (bl === 0) h.status = 1;
        if (h.status !== 0) if (h.status !== 1) return -1;
        tl = h.root;
        bl = h.m;
        for (i = 0; i < nd; i++) ll[i] = ll[i + nl];
        bd = D_BITS;
        h = new Build(ll, nd, 0, COPY_DIST, COPY_DEXT, bd);
        td = h.root;
        bd = h.m;
        if (bd === 0 && nl > 257) return -1;
        if (h.status !== 0) return -1;
        return inflate_codes(buff, off, size);
    }
    let inflate_start = () => {
        if (!slide) slide = [];
        wp = 0;
        bit_buf = 0;
        bit_len = 0;
        method = -1;
        eof = false;
        copy_leng = copy_dist = 0;
        tl = null;
    }
    let inflate_internal = (buff, off, size) => {
        let i;
        let n = 0;
        while (n < size) {
            if (eof && method === -1) return n;
            if (copy_leng > 0) {
                if (method !== STORED_BLOCK) {
                    while (copy_leng > 0 && n < size) {
                        copy_leng--;
                        copy_dist &= W_SIZE - 1;
                        wp &= W_SIZE - 1;
                        buff[off + n++] = slide[wp++] = slide[copy_dist++];
                    }
                } else {
                    while (copy_leng > 0 && n < size) {
                        copy_leng--;
                        wp &= W_SIZE - 1;
                        NEED_BITS(8);
                        buff[off + n++] = slide[wp++] = GET_BITS(8);
                        DUMP_BITS(8);
                    }
                    if (copy_leng === 0) method = -1;
                }
                if (n === size) return n;
            }
            if (method === -1) {
                if (eof) break;
                NEED_BITS(1);
                if (GET_BITS(1) !== 0) eof = true;
                DUMP_BITS(1);
                NEED_BITS(2);
                method = GET_BITS(2);
                DUMP_BITS(2);
                tl = null;
                copy_leng = 0;
            }
            switch (method) {
                case STORED_BLOCK:
                    i = inflate_stored(buff, off + n, size - n);
                    break;
                case STATIC_TREES:
                    if (tl) i = inflate_codes(buff, off + n, size - n);
                    else i = inflate_fixed(buff, off + n, size - n);
                    break;
                case DYN_TREES:
                    if (tl) i = inflate_codes(buff, off + n, size - n);
                    else i = inflate_dynamic(buff, off + n, size - n);
                    break;
                default:
                    i = -1;
                    break;
            }
            if (i === -1) {
                if (eof) return 0;
                return -1;
            }
            n += i;
        }
        return n;
    }
    let key1 = ENCODING.indexOf(input.charAt(input.length-1));
    let key2 = ENCODING.indexOf(input.charAt(input.length-2));
    let bytes = (input.length/4) * 3;
    if (key1 === 64) bytes--;
    if (key2 === 64) bytes--;
    let chr1, chr2, chr3;
    let enc1, enc2, enc3, enc4;
    let i = 0;
    let j = 0;
    let arr = new Uint8Array(bytes);
    input = input.replace(/[^A-Za-z0-9\+\/\=]/g, "");
    for (i=0; i<bytes; i+=3) {
        enc1 = ENCODING.indexOf(input.charAt(j++));
        enc2 = ENCODING.indexOf(input.charAt(j++));
        enc3 = ENCODING.indexOf(input.charAt(j++));
        enc4 = ENCODING.indexOf(input.charAt(j++));
        chr1 = (enc1 << 2) | (enc2 >> 4);
        chr2 = ((enc2 & 15) << 4) | (enc3 >> 2);
        chr3 = ((enc3 & 3) << 6) | enc4;
        arr[i] = chr1;
        if (enc3 !== 64) arr[i+1] = chr2;
        if (enc4 !== 64) arr[i+2] = chr3;
    }
    let buf = [], buf_i;
    inflate_start();
    inflate_data = arr;
    inflate_pos = 0;
    do buf_i = inflate_internal(buf, buf.length, 1024);
    while (buf_i > 0);
    inflate_data = null;
    return new Uint8Array(buf).buffer;
})(` + "`" + base64.StdEncoding.EncodeToString(buf.Bytes()) + "`" + `)`, nil
}

func wrap(name string, wasm string) (string, error) {
	cmd := exec.Command("go", "version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	goVersion := strings.Replace(string(out), "\n", "", -1)
	goVersion = strings.Replace(goVersion, "\r", "", -1)
	goVersion = strings.TrimSpace(goVersion)
	goVersion = strings.TrimPrefix(goVersion, "go version ")
	cmd = exec.Command("go", "env", "GOROOT")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	goRoot := strings.Replace(string(out), "\n", "", -1)
	goRoot = strings.Replace(goRoot, "\r", "", -1)
	goRoot = strings.TrimSpace(goRoot)
	if string(goRoot) == "" {
		return "", fmt.Errorf("GOROOT is empty")
	}
	for _, dir := range []string{"lib", "misc"} {
		if code, err := os.ReadFile(filepath.Join(goRoot, dir, "wasm", "wasm_exec.js")); err == nil {
			pre := ""
			post := ""
			if name != "" {
				pre = "globalThis[\"" + name + "\"] = (env, args) => {\n" +
					"if(env && typeof env === 'object') go.env = env;\n" +
					"if(args && args.length > 0) go.argv.push(args);\n"
				post = "}\n"
			}
			code = bytes.Replace(code, []byte("globalThis.Go ="), []byte("\tif(globalThis[\""+name+"\"]) throw new Error('global function \""+name+"\" already exists');\n"+
				pre+
				"const go = new "), 1)
			code = bytes.Replace(code, []byte("})();"), []byte("WebAssembly.instantiate("+wasm+", go.importObject).then(({instance}) => {\n"+
				"go.run(instance);\n"+
				"})\n"+
				post+
				"})();\n"), 1)
			m := minify.New()
			m.AddFunc("application/javascript", js.Minify)
			buf := &bytes.Buffer{}
			buf.WriteString("// built with " + goVersion + " at " + time.Now().Format(time.RFC3339) + "\n")
			if err := m.Minify("application/javascript", buf, bytes.NewReader(code)); err != nil {
				return "", err
			}
			return buf.String(), nil
		}
	}
	return "", fmt.Errorf("wasm_exec.js not found in %s", goRoot)
}

func build(path string) ([]byte, error) {
	out := filepath.Join(os.TempDir(), "wasmpack"+strconv.Itoa(time.Now().Nanosecond())+".wasm")
	cmd := exec.Command("go", "build", "-o", out, "-ldflags=-s -w", "-trimpath", path)
	cmd.Env = append([]string{"GOOS=js", "GOARCH=wasm", "CGO_ENABLED=0"}, os.Environ()...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("error building wasm: %w: %s", err, string(out))
	}
	wasm, err := os.ReadFile(out)
	if err != nil {
		return nil, fmt.Errorf("error reading wasm file: %w", err)
	}
	err = os.Remove(out)
	if err != nil {
		return nil, fmt.Errorf("error removing wasm file: %w", err)
	}
	return wasm, nil
}
