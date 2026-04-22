 Admin Panel                                                                                                           
                                                                                                                        
  http://163.245.219.121/admin/panel                                                                                    
                                                                                                                      
  Login prompt will appear — username admin, password whatever you set with:                                            
  config admin_password YourPassword
                                                                                                                        
  From there you can:                                                                                                   
  - Create users
  - See all lures + sessions                                                                                            
  - Get user panel URLs                                                                                               
                                                                                                                        
  ---
  User Panel                                                                                                            
                                                                                                                        
  Each user gets their own link:
  http://163.245.219.121/panel/<token>                                                                                  
                                                                                                                      
  Get a user's token inside x-tymus terminal:
  user token <username>                                                                                                 
   
  It will print the full panel URL.                                                                                     
                                                                                                                      
  ---                                                                                                                   
  If you can't reach port 80                                                                                          
                                                                                                                        
  Make sure the HTTP server is reachable:
                                                                                                                        
  # On the server                                                                                                     
  ss -tulpn | grep :80                                                                                                  
                                                                                                                      
  # From your machine
  curl http://163.245.219.121/admin/panel
                                                                                                                        
  If nothing is on port 80, x-tymus didn't start properly. Check logs:                                                  
  tail -f /tmp/x-tymus.log                                                                                              
                                            