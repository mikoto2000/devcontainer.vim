if !has("nvim")
  nnoremap <silent> "*yy yy:call SendToCdr('"')<CR>
  vnoremap <silent> "*y y:call SendToCdr('"')<CR>
else
  nnoremap <silent> "*yy yy:lua SendToCdr('"')<CR>
  vnoremap <silent> "*y y:lua SendToCdr('"')<CR>
endif
