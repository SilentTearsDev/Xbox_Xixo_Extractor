import tkinter as tk
from tkinter import filedialog, scrolledtext, messagebox
import subprocess
import threading
import os
import shutil

REPO_URL = "https://github.com/XboxDev/extract-xiso.git"
REPO_DIR = "extract-xiso"


class XboxExtractor:

    def __init__(self, root):
        self.root = root
        self.root.title("Xbox ISO Extractor")
        self.root.geometry("900x600")

        top = tk.Frame(root)
        top.pack(fill="x", padx=10, pady=10)

        self.iso_entry = tk.Entry(top)
        self.iso_entry.pack(side="left", fill="x", expand=True)

        browse_btn = tk.Button(
            top,
            text="Browse ISO",
            command=self.browse_iso
        )
        browse_btn.pack(side="left", padx=5)

        extract_btn = tk.Button(
            root,
            text="Install + Extract",
            command=self.start
        )
        extract_btn.pack(pady=5)

        self.output = scrolledtext.ScrolledText(root)
        self.output.pack(fill="both", expand=True, padx=10, pady=10)

    def log(self, text):
        self.output.insert(tk.END, text)
        self.output.see(tk.END)
        self.root.update_idletasks()

    def browse_iso(self):
        filename = filedialog.askopenfilename(
            title="Select Xbox ISO",
            filetypes=[("ISO Files", "*.iso"), ("All Files", "*.*")]
        )

        if filename:
            self.iso_entry.delete(0, tk.END)
            self.iso_entry.insert(0, filename)

    def run_command(self, cmd, cwd=None):

        self.log(f"\n$ {' '.join(cmd)}\n\n")

        process = subprocess.Popen(
            cmd,
            cwd=cwd,
            stdout=subprocess.PIPE,
            stderr=subprocess.STDOUT,
            text=True
        )

        for line in process.stdout:
            self.log(line)

        process.wait()

        if process.returncode != 0:
            raise RuntimeError(
                f"Command failed: {' '.join(cmd)}"
            )

    def install_extract_xiso(self):

        binary = os.path.join(REPO_DIR, "build", "extract-xiso")

        if os.path.exists(binary):
            self.log("\nextract-xiso already installed.\n")
            return os.path.abspath(binary)

        self.log("\n=== INSTALLING extract-xiso ===\n")

        if not os.path.exists(REPO_DIR):
            self.run_command([
                "git",
                "clone",
                REPO_URL
            ])

        build_dir = os.path.join(REPO_DIR, "build")

        os.makedirs(build_dir, exist_ok=True)

        self.run_command(["cmake", ".."],cwd=build_dir)

        self.run_command(["make"], cwd=build_dir)

        if not os.path.exists(binary):
            raise RuntimeError("Build completed but extract-xiso was not found.")

        return os.path.abspath(binary)

    def extract_iso(self, binary, iso_path):

        self.log("\n=== EXTRACTING ISO ===\n")

        iso_dir = os.path.dirname(iso_path)
        iso_name = os.path.splitext(os.path.basename(iso_path))[0]

        extracted_folder = os.path.join(iso_dir,f"{iso_name} [Extracted]")

        os.makedirs(extracted_folder,exist_ok=True)

        self.log(
            f"\nOutput folder:\n{extracted_folder}\n")

        before = set(os.listdir(iso_dir))

        self.run_command([binary, "-x", iso_path],cwd=iso_dir)

        after = set(os.listdir(iso_dir))
        new_items = after - before

        for item in new_items:

            src = os.path.join(iso_dir,item)

            if src == extracted_folder:
                continue

            try:
                shutil.move(
                    src,
                    extracted_folder
                )
            except Exception as e:
                self.log(
                    f"Could not move {item}: {e}\n"
                )

        self.log("\n=== DONE ===\n")

        subprocess.Popen(
            ["xdg-open", extracted_folder]
        )

    def worker(self):

        try:

            iso_path = self.iso_entry.get()

            if not iso_path:
                raise RuntimeError(
                    "Please select an ISO first."
                )

            binary = self.install_extract_xiso()

            self.extract_iso(
                binary,
                iso_path
            )

            messagebox.showinfo(
                "Finished",
                "Extraction completed!"
            )

        except Exception as e:

            self.log(
                f"\n\nERROR:\n{e}\n"
            )

            messagebox.showerror(
                "Error",
                str(e)
            )

    def start(self):
        threading.Thread(
            target=self.worker,
            daemon=True
        ).start()


root = tk.Tk()
app = XboxExtractor(root)
root.mainloop()